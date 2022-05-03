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
	"crypto"
	"fmt"
	"log"
	"time"

	jvsapi "github.com/abcxyz/jvs/api/v0"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Processor performs the necessary logic to validate a justification, then mints a token.
type Processor struct {
	jvspb.UnimplementedJVSServiceServer
	Signer crypto.Signer
}

const jvsIssuer = "jvs-service"

// CreateToken implements the create token API which creates and signs a JWT token if the provided justifications
// are valid.
func (p *Processor) CreateToken(ctx context.Context, request *jvspb.CreateJustificationRequest) (string, error) {
	if err := p.runValidations(request); err != nil {
		log.Printf("Couldn't validate request: %v\n", err)
		return "", status.Error(codes.InvalidArgument, "couldn't validate request")
	}
	token := p.createToken(ctx, request)
	signedToken, err := jvscrypto.SignToken(token, p.Signer)
	if err != nil {
		log.Printf("Ran into error while signing: %v\n", err)
		return "", status.Error(codes.Internal, "ran into error while minting token")
	}

	return signedToken, nil
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

// create a key with the correct claims and sign it using KMS key
func (p *Processor) createToken(ctx context.Context, request *jvspb.CreateJustificationRequest) *jwt.Token {
	now := time.Now().UTC()
	claims := &jvsapi.JVSClaims{
		StandardClaims: &jwt.StandardClaims{
			Audience:  "TODO",
			ExpiresAt: now.Add(request.Ttl.AsDuration()).Unix(),
			Id:        uuid.New().String(),
			IssuedAt:  now.Unix(),
			Issuer:    jvsIssuer,
			NotBefore: now.Unix(),
			Subject:   "TODO",
		},
		Justifications: request.Justifications,
	}
	return jwt.NewWithClaims(jwt.SigningMethodES256, claims)
}
