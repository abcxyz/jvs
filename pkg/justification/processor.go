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
	"github.com/abcxyz/jvs/pkg/cache"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/jvs/pkg/zlogger"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/sethvargo/go-gcpkms/pkg/gcpkms"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Processor performs the necessary loggeric to validate a justification, then
// mints a token.
type Processor struct {
	jvspb.UnimplementedJVSServiceServer
	kms    *kms.KeyManagementClient
	config *config.JustificationConfig
	cache  *cache.Cache[*signerWithId]
}

type signerWithId struct {
	*gcpkms.Signer
	id string
}

// NewProcessor creates a processor with the signer cache initialized
func NewProcessor(kms *kms.KeyManagementClient, config *config.JustificationConfig) *Processor {
	cache := cache.New[*signerWithId](config.SignerCacheTimeout)
	return &Processor{
		kms:    kms,
		config: config,
		cache:  cache,
	}
}

const (
	cacheKey = "signer"
)

// CreateToken implements the create token API which creates and signs a JWT
// token if the provided justifications are valid.
func (p *Processor) CreateToken(ctx context.Context, request *jvspb.CreateJustificationRequest) (string, error) {
	logger := zlogger.FromContext(ctx)
	if err := p.runValidations(request); err != nil {
		logger.Error("Couldn't validate request", zap.Error(err))
		return "", status.Error(codes.InvalidArgument, "couldn't validate request")
	}
	token := p.createToken(ctx, request)

	signer, err := p.cache.WriteThruLookup(cacheKey, func() (*signerWithId, error) {
		return p.getLatestSigner(ctx)
	})
	if err != nil {
		logger.Error("Couldn't update keys from kms", zap.Error(err))
		return "", status.Error(codes.Internal, "couldn't update keys")
	}
	token.Header["kid"] = signer.id // set key id
	sig := signer.WithContext(ctx)  // add ctx to kms signer
	signedToken, err := jvscrypto.SignToken(token, sig)
	if err != nil {
		logger.Error("Ran into error while signing", zap.Error(err))
		return "", status.Error(codes.Internal, "ran into error while minting token")
	}

	return signedToken, nil
}

func (p *Processor) getLatestSigner(ctx context.Context) (*signerWithId, error) {
	ver, err := jvscrypto.GetLatestKeyVersion(ctx, p.kms, p.config.KeyName)
	if err != nil {
		return nil, fmt.Errorf("failed to get key version, %w", err)
	}
	sig, err := gcpkms.NewSigner(ctx, p.kms, ver.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer, %w", err)
	}
	signer := &signerWithId{
		Signer: sig,
		id:     ver.Name,
	}
	return signer, nil
}

// TODO: Each category should have its own validator struct, with a shared interface.
func (p *Processor) runValidations(request *jvspb.CreateJustificationRequest) error {
	if len(request.Justifications) < 1 {
		return fmt.Errorf("no justifications specified")
	}

	if request.Ttl == nil {
		return fmt.Errorf("no ttl specified")
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

// createToken creates a key with the correct claims and sign it using KMS key.
func (p *Processor) createToken(ctx context.Context, request *jvspb.CreateJustificationRequest) *jwt.Token {
	now := time.Now().UTC()
	claims := &jvspb.JVSClaims{
		StandardClaims: &jwt.StandardClaims{
			Audience:  "TODO #22",
			ExpiresAt: now.Add(request.Ttl.AsDuration()).Unix(),
			Id:        uuid.New().String(),
			IssuedAt:  now.Unix(),
			Issuer:    p.config.Issuer,
			NotBefore: now.Unix(),
			Subject:   "TODO #22",
		},
		Justifications: request.Justifications,
	}

	return jwt.NewWithClaims(jwt.SigningMethodES256, claims)
}
