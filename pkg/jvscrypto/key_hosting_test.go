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
	"net"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/abcxyz/pkg/cache"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

func TestJWKSetFormattedString(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var clientOpt option.ClientOption
	mockKMSServer := &testutil.MockKeyManagementServer{
		UnimplementedKeyManagementServiceServer: kmspb.UnimplementedKeyManagementServiceServer{},
		Reqs:                                    make([]proto.Message, 1),
		Err:                                     nil,
		Resps:                                   make([]proto.Message, 1),
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	mockKMSServer.PrivateKey = privateKey
	x509EncodedPub, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		t.Fatal(err)
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})
	mockKMSServer.PublicKey = string(pemEncodedPub)

	serv := grpc.NewServer()
	kmspb.RegisterKeyManagementServiceServer(serv, mockKMSServer)

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
		PublicKeyConfig: &config.PublicKeyConfig{},
		Cache:           cache,
	}

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
			primary: "ver_" + versionSuffix,
			numKeys: 1,
			wantOutput: fmt.Sprintf(`{"keys":[{"crv":"P-256","kid":"%s","kty":"EC","x":"%s","y":"%s"}]}`,
				key+"/cryptoKeyVersions/[VERSION]-0",
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.X.Bytes()),
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.Y.Bytes())),
		},
		{
			name:    "multi-key",
			primary: "ver_" + versionSuffix,
			numKeys: 2,
			wantOutput: fmt.Sprintf(`{"keys":[{"crv":"P-256","kid":"%s","kty":"EC","x":"%s","y":"%s"},{"crv":"P-256","kid":"%s","kty":"EC","x":"%s","y":"%s"}]}`,
				key+"/cryptoKeyVersions/[VERSION]-0",
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.X.Bytes()),
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.Y.Bytes()),
				key+"/cryptoKeyVersions/[VERSION]-1",
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.X.Bytes()),
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.Y.Bytes())),
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
			mockKMSServer.KeyName = key
			mockKMSServer.VersionName = key + "/cryptoKeyVersions/" + versionSuffix
			mockKMSServer.Labels = make(map[string]string)
			mockKMSServer.Labels["primary"] = tc.primary
			mockKMSServer.NumVersions = tc.numKeys

			keys, err := ks.jwkList(ctx, key)
			if err != nil {
				t.Error(err)
			}
			got, err := formatJWKString(keys)
			testutil.ErrCmp(t, tc.wantErr, err)

			if err != nil {
				return
			}
			if diff := cmp.Diff(tc.wantOutput, got); diff != "" {
				t.Errorf("Got diff (-want, +got): %v", diff)
			}
		})
	}
}
