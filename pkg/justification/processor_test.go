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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"reflect"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/abcxyz/pkg/grpcutil"
	pkgtestutil "github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/durationpb"
)

type MockJWTAuthHandler struct {
	grpcutil.JWTAuthenticationHandler
}

func (j *MockJWTAuthHandler) RequestPrincipal(ctx context.Context) string {
	return "me@example.com"
}

func TestCreateToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		request       *jvspb.CreateJustificationRequest
		wantAudiences []string
		wantErr       string
		serverErr     error
	}{
		{
			name: "happy_path",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "test",
					},
				},
				Ttl: durationpb.New(3600 * time.Second),
			},
			wantAudiences: []string{DefaultAudience},
		},
		{
			name: "override_aud",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "test",
					},
				},
				Ttl:       durationpb.New(3600 * time.Second),
				Audiences: []string{"aud1", "aud2"},
			},
			wantAudiences: []string{"aud1", "aud2"},
		},
		{
			name: "no_justification",
			request: &jvspb.CreateJustificationRequest{
				Ttl: durationpb.New(3600 * time.Second),
			},
			wantErr: "failed to validate request",
		},
		{
			name: "no_ttl",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "test",
					},
				},
			},
			wantErr: "failed to validate request",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			now := time.Now().UTC()

			var clientOpt option.ClientOption
			key := "projects/[PROJECT]/locations/[LOCATION]/keyRings/[KEY_RING]/cryptoKeys/[CRYPTO_KEY]"
			version := key + "/cryptoKeyVersions/[VERSION]"
			keyID := key + "/cryptoKeyVersions/[VERSION]-0"

			mockKeyManagement := testutil.NewMockKeyManagementServer(key, version, jvscrypto.PrimaryLabelPrefix+"[VERSION]"+"-0")
			mockKeyManagement.NumVersions = 1

			privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			mockKeyManagement.PrivateKey = privateKey
			x509EncodedPub, err := x509.MarshalPKIXPublicKey(privateKey.Public())
			if err != nil {
				t.Fatal(err)
			}
			pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})
			mockKeyManagement.PublicKey = string(pemEncodedPub)

			serv := grpc.NewServer()
			kmspb.RegisterKeyManagementServiceServer(serv, mockKeyManagement)

			_, conn := pkgtestutil.FakeGRPCServer(t, func(s *grpc.Server) {
				kmspb.RegisterKeyManagementServiceServer(s, mockKeyManagement)
			})

			clientOpt = option.WithGRPCConn(conn)
			t.Cleanup(func() {
				conn.Close()
			})

			c, err := kms.NewKeyManagementClient(ctx, clientOpt)
			if err != nil {
				t.Fatal(err)
			}

			authKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			ecdsaKey, err := jwk.FromRaw(authKey.PublicKey)
			if err != nil {
				t.Fatal(err)
			}

			if err := ecdsaKey.Set(jwk.KeyIDKey, keyID); err != nil {
				t.Fatal(err)
			}

			tok := pkgtestutil.CreateJWT(t, "test_id", "user@example.com")
			validJWT := pkgtestutil.SignToken(t, tok, authKey, keyID)

			ctx = metadata.NewIncomingContext(ctx, metadata.New(map[string]string{
				"authorization": "Bearer " + validJWT,
			}))

			authHandler, err := grpcutil.NewJWTAuthenticationHandler(ctx, grpcutil.NoJWTAuthValidation())
			if err != nil {
				t.Fatal(err)
			}

			processor := NewProcessor(c, &config.JustificationConfig{
				Version:            "1",
				KeyName:            key,
				SignerCacheTimeout: 5 * time.Minute,
				Issuer:             "test-iss",
			}, authHandler)

			mockKeyManagement.Reqs = nil
			mockKeyManagement.Err = tc.serverErr

			mockKeyManagement.Resps = append(mockKeyManagement.Resps[:0], &kmspb.CryptoKeyVersion{})

			response, gotErr := processor.CreateToken(ctx, tc.request)
			if diff := pkgtestutil.DiffErrString(gotErr, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if gotErr != nil {
				return
			}

			// Validate message headers - we have to parse the full envelope for this.
			message, err := jws.Parse(response)
			if err != nil {
				t.Fatal(err)
			}
			sigs := message.Signatures()
			if got, want := len(sigs), 1; got != want {
				t.Errorf("expected length %d to be %d: %#v", got, want, sigs)
			} else {
				headers := sigs[0].ProtectedHeaders()
				if got, want := headers.Type(), "JWT"; got != want {
					t.Errorf("typ: expected %q to be %q", got, want)
				}
				if got, want := string(headers.Algorithm()), "ES256"; got != want {
					t.Errorf("alg: expected %q to be %q", got, want)
				}
				if got, want := headers.KeyID(), keyID; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			}

			// Parse as a JWT.
			token, err := jwt.Parse(response,
				jwt.WithKey(jwa.ES256, privateKey.Public()),
				jvspb.WithTypedJustifications())
			if err != nil {
				t.Fatal(err)
			}

			// Validate standard claims.
			if got, want := token.Audience(), tc.wantAudiences; !reflect.DeepEqual(got, want) {
				t.Errorf("aud: expected %q to be %q", got, want)
			}
			if got := token.Expiration(); !got.After(now) {
				t.Errorf("exp: expected %q to be after %q (%q)", got, now, got.Sub(now))
			}
			if got := token.IssuedAt(); got.IsZero() {
				t.Errorf("iat: expected %q to be", got)
			}
			if got, want := token.Issuer(), "test-iss"; got != want {
				t.Errorf("iss: expected %q to be %q", got, want)
			}
			if got, want := len(token.JwtID()), 36; got != want {
				t.Errorf("jti: expected length %d to be %d: %#v", got, want, token.JwtID())
			}
			if got := token.NotBefore(); !got.Before(now) {
				t.Errorf("nbf: expected %q to be after %q (%q)", got, now, got.Sub(now))
			}
			if got, want := token.Subject(), "user@example.com"; got != want {
				t.Errorf("sub: expected %q to be %q", got, want)
			}

			// Validate custom claims.
			gotJustifications, err := jvspb.GetJustifications(token)
			if err != nil {
				t.Fatal(err)
			}
			expectedJustifications := tc.request.Justifications
			if diff := cmp.Diff(expectedJustifications, gotJustifications, cmpopts.IgnoreUnexported(jvspb.Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}
		})
	}
}
