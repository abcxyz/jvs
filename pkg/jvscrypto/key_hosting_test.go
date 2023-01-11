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

package jvscrypto

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/abcxyz/pkg/cache"
	pkgtestutil "github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func TestGenerateJWKString(t *testing.T) {
	t.Parallel()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	x509EncodedPub, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		t.Fatal(err)
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	key := "projects/[PROJECT]/locations/[LOCATION]/keyRings/[KEY_RING]/cryptoKeys/[CRYPTO_KEY]"
	versionSuffix := "[VERSION]"

	tests := []struct {
		name       string
		primary    string
		numKeys    int
		wantOutput string
		wantErr    string
	}{
		{
			name:    "happy-path",
			primary: PrimaryLabelPrefix + versionSuffix,
			numKeys: 1,
			wantOutput: fmt.Sprintf(`{"keys":[{"crv":"P-256","kid":"%s","kty":"EC","x":"%s","y":"%s"}]}`,
				key+"/cryptoKeyVersions/[VERSION]-0",
				base64.RawURLEncoding.EncodeToString(privateKey.X.Bytes()),
				base64.RawURLEncoding.EncodeToString(privateKey.Y.Bytes())),
		},
		{
			name:    "multi-key",
			primary: PrimaryLabelPrefix + versionSuffix,
			numKeys: 2,
			wantOutput: fmt.Sprintf(`{"keys":[{"crv":"P-256","kid":"%s","kty":"EC","x":"%s","y":"%s"},{"crv":"P-256","kid":"%s","kty":"EC","x":"%s","y":"%s"}]}`,
				key+"/cryptoKeyVersions/[VERSION]-0",
				base64.RawURLEncoding.EncodeToString(privateKey.X.Bytes()),
				base64.RawURLEncoding.EncodeToString(privateKey.Y.Bytes()),
				key+"/cryptoKeyVersions/[VERSION]-1",
				base64.RawURLEncoding.EncodeToString(privateKey.X.Bytes()),
				base64.RawURLEncoding.EncodeToString(privateKey.Y.Bytes())),
		},
		{
			name:       "no-primary",
			numKeys:    0,
			wantOutput: `{"keys":[]}`,
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			mockKMSServer := testutil.NewMockKeyManagementServer(key, key+"/cryptoKeyVersions/"+versionSuffix, tc.primary)
			mockKMSServer.PrivateKey = privateKey
			mockKMSServer.PublicKey = string(pemEncodedPub)
			mockKMSServer.NumVersions = tc.numKeys

			_, conn := pkgtestutil.FakeGRPCServer(t, func(s *grpc.Server) {
				kmspb.RegisterKeyManagementServiceServer(s, mockKMSServer)
			})
			clientOpt := option.WithGRPCConn(conn)
			t.Cleanup(func() {
				conn.Close()
			})

			kms, err := kms.NewKeyManagementClient(ctx, clientOpt)
			if err != nil {
				t.Fatal(err)
			}

			cache := cache.New[string](5 * time.Minute)
			if err != nil {
				t.Fatal(err)
			}

			ks := &KeyServer{
				KMSClient:       kms,
				PublicKeyConfig: &config.PublicKeyConfig{KeyNames: []string{key}},
				Cache:           cache,
			}

			got, err := ks.generateJWKString(ctx)
			if diff := pkgtestutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tc.wantOutput, got); diff != "" {
				t.Errorf("Got diff (-want, +got): %s", diff)
			}
		})
	}
}
