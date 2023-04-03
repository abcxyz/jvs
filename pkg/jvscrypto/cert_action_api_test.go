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
	"fmt"
	"testing"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/abcxyz/pkg/logging"
	pkgtestutil "github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestCertificateAction(t *testing.T) {
	t.Parallel()

	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", "[PROJECT]", "[LOCATION]", "[KEY_RING]", "[CRYPTO_KEY]")
	versionSuffix := "[VERSION]"
	versionName := fmt.Sprintf("%s/cryptoKeyVersions/%s", parent, versionSuffix)

	cases := []struct {
		name             string
		request          *jvspb.CertificateActionRequest
		priorPrimary     string
		expectedRequests []proto.Message
		expectedPrimary  string
		wantErr          string
		serverErr        error
	}{
		{
			name: "rotate",
			request: &jvspb.CertificateActionRequest{
				Actions: []*jvspb.Action{
					{
						Version: versionName,
						Action:  jvspb.Action_ROTATE,
					},
				},
			},
			priorPrimary: PrimaryLabelPrefix + versionSuffix,
			expectedRequests: []proto.Message{
				// Look up the existing key version
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName,
				},
				// Look up the existing key
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				// Create a new key
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				// Lookup the new version
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName + "-new",
				},
				// Lookup the key again
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				// Update the primary label
				&kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Labels: map[string]string{PrimaryKey: PrimaryLabelPrefix + versionSuffix + "-new"},
						Name:   parent,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				},
			},
			expectedPrimary: PrimaryLabelPrefix + versionSuffix + "-new",
			wantErr:         "",
			serverErr:       nil,
		},
		{
			name: "force_disable",
			request: &jvspb.CertificateActionRequest{
				Actions: []*jvspb.Action{
					{
						Version: versionName,
						Action:  jvspb.Action_FORCE_DISABLE,
					},
				},
			},
			priorPrimary: PrimaryLabelPrefix + versionSuffix,
			expectedRequests: []proto.Message{
				// Look up the existing key version
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName,
				},
				// Look up the existing key
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				// Create a new key
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				// Lookup the new version
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName + "-new",
				},
				// Lookup the key again
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				// Update the primary label
				&kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Labels: map[string]string{PrimaryKey: PrimaryLabelPrefix + versionSuffix + "-new"},
						Name:   parent,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				},
				// Mark the key as disabled
				&kmspb.UpdateCryptoKeyVersionRequest{
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{
						State: kmspb.CryptoKeyVersion_DISABLED,
						Name:  versionName,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"state"},
					},
				},
			},
			expectedPrimary: PrimaryLabelPrefix + versionSuffix + "-new",
			wantErr:         "",
			serverErr:       nil,
		},
		{
			name: "force_disable_non_primary",
			request: &jvspb.CertificateActionRequest{
				Actions: []*jvspb.Action{
					{
						Version: versionName + "2",
						Action:  jvspb.Action_FORCE_DISABLE,
					},
				},
			},
			priorPrimary: PrimaryLabelPrefix + versionSuffix,
			expectedRequests: []proto.Message{
				// Look up the existing key version
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName + "2",
				},
				// Look up the existing key
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				// Mark the key as disabled
				&kmspb.UpdateCryptoKeyVersionRequest{
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{
						State: kmspb.CryptoKeyVersion_DISABLED,
						Name:  versionName + "2",
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"state"},
					},
				},
			},
			expectedPrimary: PrimaryLabelPrefix + versionSuffix,
			wantErr:         "",
			serverErr:       nil,
		},
		{
			name: "force_destroy",
			request: &jvspb.CertificateActionRequest{
				Actions: []*jvspb.Action{
					{
						Version: versionName,
						Action:  jvspb.Action_FORCE_DESTROY,
					},
				},
			},
			priorPrimary: PrimaryLabelPrefix + versionSuffix,
			expectedRequests: []proto.Message{
				// Look up the existing key version
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName,
				},
				// Look up the existing key
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				// Create a new key
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				// Lookup the new version
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName + "-new",
				},
				// Lookup the key again
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				// Update the primary label
				&kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Labels: map[string]string{PrimaryKey: PrimaryLabelPrefix + versionSuffix + "-new"},
						Name:   parent,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				},
				// Destroy the old version
				&kmspb.DestroyCryptoKeyVersionRequest{
					Name: versionName,
				},
			},
			expectedPrimary: PrimaryLabelPrefix + versionSuffix + "-new",
			wantErr:         "",
			serverErr:       nil,
		},
		{
			name: "multi_action",
			request: &jvspb.CertificateActionRequest{
				Actions: []*jvspb.Action{
					{
						Version: versionName + "2",
						Action:  jvspb.Action_FORCE_DISABLE,
					},
					{
						Version: versionName,
						Action:  jvspb.Action_ROTATE,
					},
				},
			},
			priorPrimary: PrimaryLabelPrefix + versionSuffix,
			expectedRequests: []proto.Message{
				// Look up the existing key version
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName + "2",
				},
				// Look up the existing key
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				// Look up the existing key version
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName,
				},
				// Look up the existing key
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				// Mark the key as disabled
				&kmspb.UpdateCryptoKeyVersionRequest{
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{
						State: kmspb.CryptoKeyVersion_DISABLED,
						Name:  versionName + "2",
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"state"},
					},
				},
				// Create a new key
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				// Lookup the new version
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName + "-new",
				},
				// Look up the existing key
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Labels: map[string]string{PrimaryKey: PrimaryLabelPrefix + versionSuffix + "-new"},
						Name:   parent,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				},
			},
			expectedPrimary: PrimaryLabelPrefix + versionSuffix + "-new",
			wantErr:         "",
			serverErr:       nil,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

			mockKMS := testutil.NewMockKeyManagementServer(parent, versionName, tc.priorPrimary)
			mockKMS.Err = tc.serverErr

			serv := grpc.NewServer()
			kmspb.RegisterKeyManagementServiceServer(serv, mockKMS)

			_, conn := pkgtestutil.FakeGRPCServer(t, func(s *grpc.Server) {
				kmspb.RegisterKeyManagementServiceServer(s, mockKMS)
			})

			opt := option.WithGRPCConn(conn)
			t.Cleanup(func() {
				conn.Close()
			})

			c, err := kms.NewKeyManagementClient(ctx, opt)
			if err != nil {
				t.Fatal(err)
			}

			service := &CertificateActionService{
				Handler: NewRotationHandler(ctx, c, &config.CertRotationConfig{
					CryptoConfig: &config.CryptoConfig{},
				}),
				KMSClient: c,
			}

			_, gotErr := service.CertificateAction(ctx, tc.request)
			if diff := pkgtestutil.DiffErrString(gotErr, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if diff := cmp.Diff(tc.expectedRequests, mockKMS.Reqs, protocmp.Transform()); diff != "" {
				t.Errorf("wrong requests: diff (-want, +got): %s", diff)
			}
			if diff := cmp.Diff(tc.expectedPrimary, mockKMS.Labels["primary"]); diff != "" {
				t.Errorf("wrong primary: diff (-want, +got): %s", diff)
			}
		})
	}
}
