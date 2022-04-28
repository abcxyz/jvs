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
	"log"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	gcpjwt "github.com/someone1/gcp-jwt-go/v2"
	"google.golang.org/api/iterator"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Processor performs the necessary logic to validate a justification, then mints a token.
type Processor struct {
	Config    *config.JustificationConfig
	KmsClient *kms.KeyManagementClient
}

func (p *Processor) CreateToken(ctx context.Context, request *jvspb.CreateJustificationRequest) (string, error) {
	if err := p.runValidations(ctx, request); err != nil {
		log.Printf("Couldn't validate request: %v", err)
		return "", status.Error(codes.InvalidArgument, "couldn't validate request")
	}
	token, err := p.mintToken(ctx, request)
	if err != nil {
		log.Printf("Ran into error while signing: %v", err)
		return "", status.Error(codes.Internal, "ran into error while minting token")
	}
	return token, nil
}

// TODO: Break this out into its own class, each validator should be its own class as well, with a shared interface.
func (p *Processor) runValidations(ctx context.Context, request *jvspb.CreateJustificationRequest) error {
	if len(request.Justifications) < 1 {
		return fmt.Errorf("no justifications specified")
	}

	var err *multierror.Error
	verifications := make([]string, 0)
	for _, j := range request.Justifications {
		switch j.Category {
		case "explanation":
			if j.Value != "" {
				verifications = append(verifications, j.Category)
			} else {
				err = multierror.Append(err, fmt.Errorf("no value specified for 'explanation' category"))
			}
		default:
			err = multierror.Append(err, fmt.Errorf("unexpected justification %v unrecognized", j))
		}
	}
	return err.ErrorOrNil()
}

type CustomClaims struct {
	*jwt.StandardClaims
	Justifications []*jvspb.Justification
}

// create a key with the correct claims and sign it using KMS key
func (p *Processor) mintToken(ctx context.Context, request *jvspb.CreateJustificationRequest) (string, error) {
	claims := &CustomClaims{
		&jwt.StandardClaims{
			Audience:  "TODO",
			ExpiresAt: time.Now().Add(request.Ttl.AsDuration()).Unix(),
			Id:        uuid.New().String(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "jvs-service",
			NotBefore: time.Now().Unix(),
			Subject:   "TODO",
		},
		request.Justifications,
	}
	token := jwt.NewWithClaims(gcpjwt.SigningMethodKMSES256, claims)
	keyCtx, err := p.getKeyContext(ctx)
	if err != nil {
		return "", err
	}

	// Sign and return token
	return token.SignedString(keyCtx)
}

// Set up a context with the correct values for use in the JWT signing library
func (p *Processor) getKeyContext(ctx context.Context) (context.Context, error) {
	ver, err := p.getLatestKeyVersion(ctx)
	if err != nil {
		return nil, err
	}

	config := &gcpjwt.KMSConfig{
		KeyPath: ver.Name,
	}
	keyCtx := gcpjwt.NewKMSContext(ctx, config)
	return keyCtx, nil
}

// Look up the newest enabled key version
func (p *Processor) getLatestKeyVersion(ctx context.Context) (*kmspb.CryptoKeyVersion, error) {
	it := p.KmsClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: p.Config.KeyName,
	})

	var newestEnabledVersion *kmspb.CryptoKeyVersion
	var newestTime time.Time
	for {
		ver, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("err while reading crypto key version list: %w", err)

		}
		if ver.State == kmspb.CryptoKeyVersion_ENABLED && (newestEnabledVersion == nil || ver.CreateTime.AsTime().After(newestTime)) {
			newestEnabledVersion = ver
			newestTime = ver.CreateTime.AsTime()
		}
	}
	return newestEnabledVersion, nil
}
