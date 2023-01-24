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
	"github.com/abcxyz/pkg/grpcutil"
	"github.com/abcxyz/pkg/logging"
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
	kms         *kms.KeyManagementClient
	config      *config.JustificationConfig
	cache       *cache.Cache[*signerWithID]
	authHandler *grpcutil.JWTAuthenticationHandler
}

type signerWithID struct {
	*gcpkms.Signer
	id string
}

// NewProcessor creates a processor with the signer cache initialized.
func NewProcessor(kms *kms.KeyManagementClient, config *config.JustificationConfig, authHandler *grpcutil.JWTAuthenticationHandler) *Processor {
	cache := cache.New[*signerWithID](config.SignerCacheTimeout)
	return &Processor{
		kms:         kms,
		config:      config,
		cache:       cache,
		authHandler: authHandler,
	}
}

const (
	cacheKey = "signer"

	// DefaultAudience is the default audience used in justification tokens. It
	// can be overridden with the audiences in the justification request.
	DefaultAudience = "dev.abcxyz.jvs"
)

// CreateToken implements the create token API which creates and signs a JWT
// token if the provided justifications are valid.
func (p *Processor) CreateToken(ctx context.Context, req *jvspb.CreateJustificationRequest) ([]byte, error) {
	now := time.Now().UTC()

	logger := logging.FromContext(ctx)

	if err := p.runValidations(req); err != nil {
		// panic(err.Error())
		logger.Errorw("failed to validate request", "error", err)
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to validate request, %s", err.Error()))
	}

	token, err := p.createToken(ctx, now, req)
	if err != nil {
		logger.Errorw("failed to create token", "error", err)
		return nil, status.Error(codes.Internal, "failed to create token")
	}

	signer, err := p.cache.WriteThruLookup(cacheKey, func() (*signerWithID, error) {
		return p.getPrimarySigner(ctx)
	})
	if err != nil {
		logger.Errorw("failed to get signer", "error", err)
		return nil, status.Error(codes.Internal, "failed to get token signer")
	}

	// Build custom headers and set the "kid" as the signer ID.
	headers := jws.NewHeaders()
	if err := headers.Set(jws.KeyIDKey, signer.id); err != nil {
		logger.Errorw("failed to set kid header", "error", err)
		return nil, status.Error(codes.Internal, "failed to set token headers")
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
		return nil, fmt.Errorf("unable to determine primary, %w", err)
	}
	if primaryVer == "" {
		return nil, fmt.Errorf("no primary version found")
	}
	sig, err := gcpkms.NewSigner(ctx, p.kms, primaryVer)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer, %w", err)
	}
	return &signerWithID{
		Signer: sig,
		id:     primaryVer,
	}, nil
}

// TODO: Each category should have its own validator struct, with a shared interface.
func (p *Processor) runValidations(request *jvspb.CreateJustificationRequest) error {
	if len(request.Justifications) < 1 {
		return fmt.Errorf("no justifications specified")
	}

	if request.Ttl == nil {
		return fmt.Errorf("no ttl specified")
	}

	if request.Ttl.AsDuration() > 24*time.Hour {
		return fmt.Errorf("token ttl shouldn't exceed 24 hours")
	}

	if request.Ttl.AsDuration() <= 0*time.Second {
		return fmt.Errorf("token ttl should be positive")
	}

	var err *multierror.Error
	for _, j := range request.Justifications {
		switch j.Category {
		case "explanation":
			if j.Value == "" {
				err = multierror.Append(err, fmt.Errorf("no value specified for 'explanation' category"))
			}
		default:
			err = multierror.Append(err, fmt.Errorf("unexpected justification %v unrecognized", j))
		}
	}
	return err.ErrorOrNil()
}

// createToken is an internal helper for testing that builds an unsigned jwt
// token from the request.
func (p *Processor) createToken(ctx context.Context, now time.Time, req *jvspb.CreateJustificationRequest) (jwt.Token, error) {
	email, err := p.authHandler.RequestPrincipal(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get email of requestor: %w", err)
	}

	id := uuid.New().String()
	exp := now.Add(req.Ttl.AsDuration())

	justs := req.Justifications

	// Use audiences in the request if provided.
	aud := req.Audiences
	if len(aud) == 0 {
		aud = []string{DefaultAudience}
	}

	token, err := jwt.NewBuilder().
		Audience(aud).
		Expiration(exp).
		IssuedAt(now).
		Issuer(p.config.Issuer).
		JwtID(id).
		NotBefore(now).
		Subject(email).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build jwt: %w", err)
	}

	if err := jvspb.SetJustifications(token, justs); err != nil {
		return nil, fmt.Errorf("failed to set justifications on jwt: %w", err)
	}

	return token, nil
}
