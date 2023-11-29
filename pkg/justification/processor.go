// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package justification

import (
	"context"
	"errors"
	"fmt"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/sethvargo/go-gcpkms/pkg/gcpkms"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/pkg/cache"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/timeutil"
)

// Processor performs the necessary logic to validate a justification, then
// mints a token.
type Processor struct {
	jvspb.UnimplementedJVSServiceServer
	kms        *kms.KeyManagementClient
	config     *config.JustificationConfig
	cache      *cache.Cache[*signerWithID]
	validators map[string]jvspb.Validator
}

type signerWithID struct {
	*gcpkms.Signer
	id string
}

// NewProcessor creates a processor with the signer cache initialized.
func NewProcessor(kms *kms.KeyManagementClient, config *config.JustificationConfig) *Processor {
	cache := cache.New[*signerWithID](config.SignerCacheTimeout)
	return &Processor{
		kms:    kms,
		config: config,
		cache:  cache,
		validators: map[string]jvspb.Validator{
			jvspb.DefaultJustificationCategory: jvspb.DefaultJustificationValidator,
		},
	}
}

const (
	cacheKey = "signer"

	// DefaultAudience is the default audience used in justification tokens. It
	// can be overridden with the audiences in the justification request.
	DefaultAudience = "dev.abcxyz.jvs"
)

// WithValidators adds validators to the processor.
func (p *Processor) WithValidators(v map[string]jvspb.Validator) *Processor {
	for k, validator := range v {
		p.validators[k] = validator
	}
	return p
}

// Validators returns all the validators allowed by this processor.
func (p *Processor) Validators() map[string]jvspb.Validator {
	return p.validators
}

// CreateToken implements the create token API which creates and signs a JWT
// token if the provided justifications are valid.
func (p *Processor) CreateToken(ctx context.Context, requestor string, req *jvspb.CreateJustificationRequest) ([]byte, error) {
	now := time.Now().UTC()

	logger := logging.FromContext(ctx)

	if err := p.runValidations(ctx, req); err != nil {
		return nil, err
	}

	token, err := p.createToken(ctx, requestor, req, now)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create token", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create token: %s", err)
	}

	signer, err := p.cache.WriteThruLookup(cacheKey, func() (*signerWithID, error) {
		return p.getPrimarySigner(ctx)
	})
	if err != nil {
		logger.ErrorContext(ctx, "failed to get token signer", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get token signer: %s", err)
	}

	// Build custom headers and set the "kid" as the signer ID.
	headers := jws.NewHeaders()
	if err := headers.Set(jws.KeyIDKey, signer.id); err != nil {
		logger.ErrorContext(ctx, "failed to set kid header", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to set token headers: %s", err)
	}

	// Sign the token.
	b, err := jwt.Sign(token, jwt.WithKey(jwa.ES256, signer, jws.WithProtectedHeaders(headers)))
	if err != nil {
		logger.ErrorContext(ctx, "failed to sign token", "error", err)
		return nil, status.Error(codes.Internal, "failed to sign token")
	}

	return b, nil
}

func (p *Processor) getPrimarySigner(ctx context.Context) (*signerWithID, error) {
	primaryVer, err := jvscrypto.GetPrimary(ctx, p.kms, p.config.KeyName)
	if err != nil {
		return nil, fmt.Errorf("failed to determine primary signing key: %w", err)
	}
	if primaryVer == "" {
		return nil, fmt.Errorf("no primary version found")
	}
	sig, err := gcpkms.NewSigner(ctx, p.kms, primaryVer)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}
	return &signerWithID{
		Signer: sig,
		id:     primaryVer,
	}, nil
}

// runValidations is an internal helper function that validates requests.
// If any errors occur during validation, it returns a standard internal error message with codes.Internal.
// If the request fails validation, it returns full error messages with codes.InvalidArgument.
func (p *Processor) runValidations(ctx context.Context, req *jvspb.CreateJustificationRequest) error {
	logger := logging.FromContext(ctx)
	if len(req.Justifications) < 1 {
		return status.Errorf(codes.InvalidArgument, "failed to validate request: no justifications specified")
	}

	var validationErr, internalErr error

	var justificationsLength int
	for _, j := range req.Justifications {
		justificationsLength += len(j.Category) + len(j.Value)

		v, ok := p.validators[j.Category]
		if !ok {
			validationErr = errors.Join(validationErr, fmt.Errorf("category %q is not supported", j.Category))
			continue
		}
		resp, verr := v.Validate(ctx, &jvspb.ValidateJustificationRequest{
			Justification: j,
		})
		if verr != nil {
			internalErr = errors.Join(internalErr, fmt.Errorf("unexpected error from validator %q: %w", j.Category, verr))
			continue
		}

		if !resp.Valid {
			validationErr = errors.Join(validationErr,
				fmt.Errorf("failed validation criteria with error %v and warning %v", resp.Error, resp.Warning))
		}

		j.Annotation = resp.Annotation
	}

	// This isn't perfect, but it's the easiest place to get "close" to limiting
	// the size.
	if got, max := justificationsLength, 4_000; got > max {
		validationErr = errors.Join(validationErr, fmt.Errorf("justification size (%d bytes) must be less than %d bytes",
			got, max))
	}

	var audiencesLength int
	for _, v := range req.Audiences {
		audiencesLength += len(v)
	}
	if got, max := audiencesLength, 1_000; got > max {
		validationErr = errors.Join(validationErr, fmt.Errorf("audiences size (%d bytes) must be less than %d bytes",
			got, max))
	}

	// In case of internal errors, a standard internal error message will be shown to the user,
	// even if there are validation errors. The complete internal error message will be logged along
	// with any additional validation errors.
	if internalErr != nil {
		logger.ErrorContext(ctx, "internal error during validation", "error", internalErr, "validation_error", validationErr)
		return status.Errorf(codes.Internal, "unable to validate request")
	}

	if validationErr != nil {
		logger.WarnContext(ctx, "failed to validate token", "error", validationErr)
		return status.Errorf(codes.InvalidArgument, "failed to validate request: %v", validationErr)
	}
	return nil
}

// createToken is an internal helper for testing that builds an unsigned jwt
// token from the request.
func (p *Processor) createToken(ctx context.Context, requestor string, req *jvspb.CreateJustificationRequest, now time.Time) (jwt.Token, error) {
	ttl, err := computeTTL(req.Ttl.AsDuration(), p.config.DefaultTTL, p.config.MaxTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to compute ttl: %w", err)
	}

	id := uuid.New().String()
	exp := now.Add(ttl)
	justs := req.Justifications
	iss := p.config.Issuer

	// Use audiences in the request if provided.
	aud := req.Audiences
	if len(aud) == 0 {
		aud = []string{DefaultAudience}
	}

	// If no subject was given, default to the caller's identity.
	subject := req.Subject
	if subject == "" {
		subject = requestor
	}

	token, err := jwt.NewBuilder().
		Audience(aud).
		Expiration(exp).
		IssuedAt(now).
		Issuer(iss).
		JwtID(id).
		NotBefore(now).
		Subject(subject).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build jwt: %w", err)
	}

	if err := jvspb.SetRequestor(token, requestor); err != nil {
		return nil, fmt.Errorf("failed to set requestor on jwt: %w", err)
	}

	if err := jvspb.SetJustifications(token, justs); err != nil {
		return nil, fmt.Errorf("failed to set justifications on jwt: %w", err)
	}

	return token, nil
}

// computeTTL is a helper that computes the best TTL given the requested TTL,
// default TTL, and maximum configured TTL. If the requested TTL is greater than
// the maximum TTL, it returns an error. If the requested TTL is 0, it returns
// the default TTL.
func computeTTL(req, def, max time.Duration) (time.Duration, error) {
	if req <= 0 {
		return def, nil
	}

	if req > max {
		return 0, fmt.Errorf("requested ttl (%s) cannot be greater than max tll (%s)",
			timeutil.HumanDuration(req), timeutil.HumanDuration(max))
	}

	return req, nil
}
