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
	"encoding/json"
	"encoding/pem"
	"net"
	"testing"
	"time"

	"github.com/abcxyz/jvs/pkg/firestore"

	kms "cloud.google.com/go/kms/apiv1"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/abcxyz/pkg/grpcutil"
	pkgtestutil "github.com/abcxyz/pkg/testutil"
	"github.com/golang-jwt/jwt"
	"github.com/google/go-cmp/cmp"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

type MockJWTAuthHandler struct {
	grpcutil.JWTAuthenticationHandler
}

func (j *MockJWTAuthHandler) RequestPrincipal(ctx context.Context) string {
	return "me@example.com"
}

type MockFirestoreRemoteConfig struct {
	config []byte
}

func (m MockFirestoreRemoteConfig) GetRemoteConfigTo(ctx context.Context, data interface{}) error {
	return json.Unmarshal(m.config, data)
}

func (m MockFirestoreRemoteConfig) UpdateRemoteConfig(ctx context.Context, configPath string, value interface{}) error {
	return nil
}

func TestCreateToken(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var clientOpt option.ClientOption
	mockKeyManagement := &testutil.MockKeyManagementServer{
		UnimplementedKeyManagementServiceServer: kmspb.UnimplementedKeyManagementServiceServer{},
		Reqs:                                    make([]proto.Message, 1),
		Err:                                     nil,
		Resps:                                   make([]proto.Message, 1),
		NumVersions:                             1,
	}

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

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	// not checked, but makes linter happy
	errs := make(chan error, 1)
	go func() {
		errs <- serv.Serve(lis)
		close(errs)
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	clientOpt = option.WithGRPCConn(conn)
	t.Cleanup(func() {
		conn.Close()
	})

	c, err := kms.NewKeyManagementClient(ctx, clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	key := "projects/[PROJECT]/locations/[LOCATION]/keyRings/[KEY_RING]/cryptoKeys/[CRYPTO_KEY]"
	version := key + "[VERSION]"
	mockKeyManagement.VersionName = version

	authKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecdsaKey, err := jwk.FromRaw(authKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	keyID := key + "/cryptoKeyVersions/[VERSION]-0"
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
	kmsJustificationConfig := firestore.KMSJustificationConfig{KeyName: key}
	configBytes, err := json.Marshal(kmsJustificationConfig)
	if err != nil {
		t.Fatal(err)
	}
	remoteConfig := MockFirestoreRemoteConfig{config: configBytes}

	processor := NewProcessor(c, remoteConfig, &config.JustificationConfig{
		FirestoreProjectID: "fakeProject",
		Version:            "1",
		SignerCacheTimeout: 5 * time.Minute,
		Issuer:             "test-iss",
	}, authHandler)

	hour, err := time.ParseDuration("3600s")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		request   *jvspb.CreateJustificationRequest
		wantErr   string
		serverErr error
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
				Ttl: durationpb.New(hour),
			},
		},
		{
			name: "no_justification",
			request: &jvspb.CreateJustificationRequest{
				Ttl: durationpb.New(hour),
			},
			wantErr: "couldn't validate request",
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
			wantErr: "couldn't validate request",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
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
			if err := jvscrypto.VerifyJWTString(ctx, c, "keyName", response); err != nil {
				t.Errorf("Unable to verify signed jwt. %v", err)
			}

			claims := &jvspb.JVSClaims{}
			token, err := jwt.ParseWithClaims(response, claims, func(token *jwt.Token) (interface{}, error) {
				return privateKey.Public(), nil
			})
			if err != nil {
				t.Errorf("Unable to parse created jwt string. %v", err)
			}
			validateClaims(t, claims, tc.request.Justifications)
			got := token.Header["kid"]
			want := version + "-0"
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Got diff (-want, +got): %v", diff)
			}
		})
	}
}

func validateClaims(tb testing.TB, provided *jvspb.JVSClaims, expectedJustifications []*jvspb.Justification) {
	tb.Helper()

	// test the standard claims filled by processor
	if got, want := provided.Issuer, "test-iss"; got != want {
		tb.Errorf("audience value %s incorrect, expected %s", got, want)
	}
	if got, want := provided.Subject, "user@example.com"; got != want {
		tb.Errorf("subject value %s incorrect, expected %s", got, want)
	}
	// TODO: as we add more standard claims, add more validations.

	if len(provided.Justifications) != len(expectedJustifications) {
		tb.Errorf("Number of justifications was incorrect.\n got: %v\n want: %v", provided.Justifications, expectedJustifications)
	}

	for _, j := range provided.Justifications {
		found := false
		for i, expectedJ := range expectedJustifications {
			if j.Value == expectedJ.Value && j.Category == expectedJ.Category {
				expectedJustifications = append(expectedJustifications[:i], expectedJustifications[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			tb.Errorf("Justifications didn't match.\n got: %v\n want: %v", provided.Justifications, expectedJustifications)
			return
		}
	}
}
