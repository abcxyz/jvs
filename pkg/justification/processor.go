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
	"fmt"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/pkg/cache"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/timeutil"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/sethvargo/go-gcpkms/pkg/gcpkms"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	}
}

const (
	cacheKey = "signer"

	// DefaultAudience is the default audience used in justification tokens. It
	// can be overridden with the audiences in the justification request.
	DefaultAudience = "dev.abcxyz.jvs"
)

func (p *Processor) WithValidators(v map[string]jvspb.Validator) *Processor {
	p.validators = v
	return p
}

// CreateToken implements the create token API which creates and signs a JWT
// token if the provided justifications are valid.
func (p *Processor) CreateToken(ctx context.Context, requestor string, req *jvspb.CreateJustificationRequest) ([]byte, error) {
	now := time.Now().UTC()

	logger := logging.FromContext(ctx)

	if err := p.runValidations(ctx, req); err != nil {
		logger.Errorw("failed to validate request", "error", err)
		return nil, status.Errorf(codes.InvalidArgument, "failed to validate request: %s", err)
	}

	token, err := p.createToken(ctx, requestor, req, now)
	if err != nil {
		logger.Errorw("failed to create token", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to create token: %s", err)
	}

	signer, err := p.cache.WriteThruLookup(cacheKey, func() (*signerWithID, error) {
		return p.getPrimarySigner(ctx)
	})
	if err != nil {
		logger.Errorw("failed to get token signer", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get token signer: %s", err)
	}

	// Build custom headers and set the "kid" as the signer ID.
	headers := jws.NewHeaders()
	if err := headers.Set(jws.KeyIDKey, signer.id); err != nil {
		logger.Errorw("failed to set kid header", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to set token headers: %s", err)
	}

	// Sign the token.
	b, err := jwt.Sign(token, jwt.WithKey(jwa.ES256, signer, jws.WithProtectedHeaders(headers)))
	if err != nil {
		logger.Errorw("failed to sign token", "error", err)
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

// TODO: Each category should have its own validator struct, with a shared interface.
func (p *Processor) runValidations(ctx context.Context, req *jvspb.CreateJustificationRequest) error {
	if len(req.Justifications) < 1 {
		return fmt.Errorf("no justifications specified")
	}

	var justificationsLength int
	var err *multierror.Error
	for _, j := range req.Justifications {
		justificationsLength += len(j.Category) + len(j.Value)

		switch j.Category {
		case "explanation":
			if j.Value == "" {
				err = multierror.Append(err, fmt.Errorf("no value specified for 'explanation' category"))
			}
		default:
			v, ok := p.validators[j.Category]
			if !ok {
				err = multierror.Append(err, fmt.Errorf("missing validator for category %q", j.Category))
				continue
			}
			resp, e := v.Validate(ctx, &jvspb.ValidateJustificationRequest{
				Justification: j,
			})
			if e != nil {
				err = multierror.Append(err, fmt.Errorf("unexpected error from validator %q: %w", j.Category, e))
				continue
			}

			if !resp.Valid {
				err = multierror.Append(err,
					fmt.Errorf("failed validation criteria with error %v and warning %v", resp.Error, resp.Warning))
			}
		}
	}

	// This isn't perfect, but it's the easiest place to get "close" to limiting
	// the size.
	if got, max := justificationsLength, 4_000; got > max {
		err = multierror.Append(err, fmt.Errorf("justification size (%d bytes) must be less than %d bytes",
			got, max))
	}

	var audiencesLength int
	for _, v := range req.Audiences {
		audiencesLength += len(v)
	}
	if got, max := audiencesLength, 1_000; got > max {
		err = multierror.Append(err, fmt.Errorf("audiences size (%d bytes) must be less than %d bytes",
			got, max))
	}

	return err.ErrorOrNil()
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
