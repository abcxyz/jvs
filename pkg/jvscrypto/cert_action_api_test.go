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
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/testutil"
	pkgtestutil "github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestCertificateAction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", "[PROJECT]", "[LOCATION]", "[KEY_RING]", "[CRYPTO_KEY]")
	versionSuffix := "[VERSION]"
	versionName := fmt.Sprintf("%s/cryptoKeyVersions/%s", parent, versionSuffix)

	tests := []struct {
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
			priorPrimary: "ver_" + versionSuffix,
			expectedRequests: []proto.Message{
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName,
				},
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Labels: map[string]string{"primary": "ver_" + versionSuffix + "-new"},
						Name:   parent,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				},
			},
			expectedPrimary: "ver_" + versionSuffix + "-new",
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
			priorPrimary: "ver_" + versionSuffix,
			expectedRequests: []proto.Message{
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName,
				},
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Labels: map[string]string{"primary": "ver_" + versionSuffix + "-new"},
						Name:   parent,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				},
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
			expectedPrimary: "ver_" + versionSuffix + "-new",
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
			priorPrimary: "ver_" + versionSuffix,
			expectedRequests: []proto.Message{
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName + "2",
				},
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
			expectedPrimary: "ver_" + versionSuffix,
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
			priorPrimary: "ver_" + versionSuffix,
			expectedRequests: []proto.Message{
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName,
				},
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Labels: map[string]string{"primary": "ver_" + versionSuffix + "-new"},
						Name:   parent,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				},
				&kmspb.DestroyCryptoKeyVersionRequest{
					Name: versionName,
				},
			},
			expectedPrimary: "ver_" + versionSuffix + "-new",
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
			priorPrimary: "ver_" + versionSuffix,
			expectedRequests: []proto.Message{
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName + "2",
				},
				&kmspb.UpdateCryptoKeyVersionRequest{
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{
						State: kmspb.CryptoKeyVersion_DISABLED,
						Name:  versionName + "2",
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"state"},
					},
				},
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName,
				},
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Labels: map[string]string{"primary": "ver_" + versionSuffix + "-new"},
						Name:   parent,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				},
			},
			expectedPrimary: "ver_" + versionSuffix + "-new",
			wantErr:         "",
			serverErr:       nil,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
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

			handler := &RotationHandler{
				KMSClient:    c,
				CryptoConfig: &config.CryptoConfig{},
			}

			service := &CertificateActionService{
				Handler:   handler,
				KMSClient: c,
			}

			gotErr := service.certificateAction(ctx, tc.request)

			if err != nil {
				t.Fatal(err)
			}

			if want, got := tc.expectedRequests, mockKMS.Reqs; !slicesEq(want, got) {
				for _, msg := range got {
					t.Errorf("got request: %s", msg)
				}
				for _, msg := range want {
					t.Errorf("want request: %s", msg)
				}
				t.Errorf("wrong requests %v, want %v", got, want)
			}
			if diff := pkgtestutil.DiffErrString(gotErr, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}

			if diff := cmp.Diff(tc.expectedPrimary, mockKMS.Labels["primary"]); diff != "" {
				t.Errorf("Got diff (-want, +got): %v", diff)
			}
		})
	}
}