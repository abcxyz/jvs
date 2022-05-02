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
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	v0 "github.com/abcxyz/jvs/api/v0"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Processor performs the necessary logic to validate a justification, then mints a token.
type Processor struct {
	Signer crypto.Signer
}

const jvs_issuer = "jvs-service"

func (p *Processor) CreateToken(ctx context.Context, request *jvspb.CreateJustificationRequest) (string, error) {
	if err := p.runValidations(request); err != nil {
		log.Printf("Couldn't validate request: %v", err)
		return "", status.Error(codes.InvalidArgument, "couldn't validate request")
	}
	token := p.createToken(ctx, request)
	signedToken, err := p.signToken(token)
	if err != nil {
		log.Printf("Ran into error while signing: %v", err)
		return "", status.Error(codes.Internal, "ran into error while minting token")
	}

	return signedToken, nil
}

// TODO: Each category should have its own validator struct, with a shared interface.
func (p *Processor) runValidations(request *jvspb.CreateJustificationRequest) error {
	if request.Justifications == nil || len(request.Justifications) < 1 {
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
	now := time.Now()
	claims := &v0.JVSClaims{
		&jwt.StandardClaims{
			Audience:  "TODO",
			ExpiresAt: now.Add(request.Ttl.AsDuration()).Unix(),
			Id:        uuid.New().String(),
			IssuedAt:  now.Unix(),
			Issuer:    jvs_issuer,
			NotBefore: now.Unix(),
			Subject:   "TODO",
		},
		request.Justifications,
	}
	return jwt.NewWithClaims(jwt.SigningMethodES256, claims)
}

func (p *Processor) signToken(token *jwt.Token) (string, error) {
	signingString, err := token.SigningString()
	if err != nil {
		return "", err
	}

	digest := sha256.Sum256([]byte(signingString))
	sig, err := p.Signer.Sign(rand.Reader, digest[:], nil)
	if err != nil {
		return "", fmt.Errorf("error signing token: %w", err)
	}

	// Unpack the ASN1 signature. ECDSA signers are supposed to return this format
	// https://golang.org/pkg/crypto/#Signer
	// All supported signers in thise codebase are verified to return ASN1.
	var parsedSig struct{ R, S *big.Int }
	// ASN1 is not the expected format for an ES256 JWT signature.
	// The output format is specified here, https://tools.ietf.org/html/rfc7518#section-3.4
	// Reproduced here for reference.
	//    The ECDSA P-256 SHA-256 digital signature is generated as follows:
	//
	// 1 .  Generate a digital signature of the JWS Signing Input using ECDSA
	//      P-256 SHA-256 with the desired private key.  The output will be
	//      the pair (R, S), where R and S are 256-bit unsigned integers.
	_, err = asn1.Unmarshal(sig, &parsedSig)
	if err != nil {
		return "", fmt.Errorf("unable to unmarshal signature: %w", err)
	}

	keyBytes := 256 / 8
	if 256%8 > 0 {
		keyBytes++
	}

	// 2. Turn R and S into octet sequences in big-endian order, with each
	// 		array being be 32 octets long.  The octet sequence
	// 		representations MUST NOT be shortened to omit any leading zero
	// 		octets contained in the values.
	rBytes := parsedSig.R.Bytes()
	rBytesPadded := make([]byte, keyBytes)
	copy(rBytesPadded[keyBytes-len(rBytes):], rBytes)

	sBytes := parsedSig.S.Bytes()
	sBytesPadded := make([]byte, keyBytes)
	copy(sBytesPadded[keyBytes-len(sBytes):], sBytes)

	// 3. Concatenate the two octet sequences in the order R and then S.
	//	 	(Note that many ECDSA implementations will directly produce this
	//	 	concatenation as their output.)
	sig = make([]byte, 0, len(rBytesPadded)+len(sBytesPadded))
	sig = append(sig, rBytesPadded...)
	sig = append(sig, sBytesPadded...)

	return strings.Join([]string{signingString, jwt.EncodeSegment(sig)}, "."), nil
}
