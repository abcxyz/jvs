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

// much of the below code is only slightly modified from https://github.com/googleapis/google-cloud-go/blob/main/kms/apiv1/mock_test.go

package jvscrypto

import (
	"context"
	"fmt"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/abcxyz/pkg/logging"
	pkgtestutil "github.com/abcxyz/pkg/testutil"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestGetKeyNameFromVersion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		input      string
		wantOutput string
		wantErr    string
	}{
		{
			name:       "happy_path",
			input:      "projects/*/locations/location1/keyRings/keyring1/cryptoKeys/key1/cryptoKeyVersions/version1",
			wantOutput: "projects/*/locations/location1/keyRings/keyring1/cryptoKeys/key1",
		},
		{
			name:       "sad_path_incorrect_number_values",
			input:      "projects/*/locations/location1/keyRings/keyring1/cryptoKeys/key1",
			wantOutput: "",
			wantErr:    "input had unexpected format: \"projects/*/locations/location1/keyRings/keyring1/cryptoKeys/key1\"",
		},
		{
			name:       "sad_path_no_slashes",
			input:      "some_weird_input",
			wantOutput: "",
			wantErr:    "input had unexpected format: \"some_weird_input\"",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			output, err := getKeyNameFromVersion(tc.input)

			if diff := cmp.Diff(tc.wantOutput, output, protocmp.Transform()); diff != "" {
				t.Errorf("Got diff (-want, +got): %v", diff)
			}

			if tc.wantErr != "" {
				if err != nil {
					if diff := cmp.Diff(err.Error(), tc.wantErr); diff != "" {
						t.Errorf("Process got unexpected error substring: %v", diff)
					}
				} else {
					t.Errorf("Expected error, but received nil")
				}
			} else if err != nil {
				t.Errorf("Expected no error, but received \"%v\"", err)
			}
		})
	}
}

func TestDetermineActions(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	keyTTL, err := time.ParseDuration("240h") // 10 days
	if err != nil {
		t.Error("Couldn't parse key ttl")
	}
	gracePeriod, err := time.ParseDuration("60m")
	if err != nil {
		t.Error("Couldn't parse grace period")
	}
	disablePeriod, err := time.ParseDuration("480h") // 20 days
	if err != nil {
		t.Error("Couldn't parse disable period")
	}
	propagationDelay, err := time.ParseDuration("12h") // half day
	if err != nil {
		t.Error("Couldn't parse propagation delay")
	}

	handler := NewRotationHandler(ctx, nil, &config.CertRotationConfig{
		KeyTTL:           keyTTL,
		GracePeriod:      gracePeriod,
		DisabledPeriod:   disablePeriod,
		PropagationDelay: propagationDelay,
	})

	curTime := time.Unix(100*60*60*24, 0) // 100 days after start

	oldEnabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 50 * 60 * 60 * 24}, // 50 days old
		State:      kmspb.CryptoKeyVersion_ENABLED,
		Name:       "oldEnabledKey",
	}
	oldEnabledKey2 := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 49 * 60 * 60 * 24}, // 51 days old
		State:      kmspb.CryptoKeyVersion_ENABLED,
		Name:       "oldEnabledKey2",
	}
	newEnabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 99 * 60 * 60 * 24}, // 1 day old
		State:      kmspb.CryptoKeyVersion_ENABLED,
		Name:       "newEnabledKey",
	}
	newEnabledKey2 := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 100 * 60 * 60 * 24}, // 0 day old
		State:      kmspb.CryptoKeyVersion_ENABLED,
		Name:       "newEnabledKey",
	}
	newDisabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 90 * 60 * 60 * 24}, // 10 days old
		State:      kmspb.CryptoKeyVersion_DISABLED,
		Name:       "newDisabledKey",
	}
	oldDisabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 1 * 60 * 60 * 24}, // 99 days old
		State:      kmspb.CryptoKeyVersion_DISABLED,
		Name:       "oldDisabledKey",
	}
	oldDestroyedKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 1 * 60 * 60 * 24}, // 99 days old
		State:      kmspb.CryptoKeyVersion_DESTROYED,
		Name:       "oldDestroyedKey",
	}

	cases := []struct {
		name        string
		versions    []*kmspb.CryptoKeyVersion
		primary     string
		wantActions []*actionTuple
		wantErr     string
	}{
		{
			name:     "no_key",
			versions: []*kmspb.CryptoKeyVersion{},
			wantActions: []*actionTuple{
				{ActionCreateNewAndPromote, nil},
			},
		},
		{
			name: "single_key_new",
			versions: []*kmspb.CryptoKeyVersion{
				newEnabledKey2,
			},
			wantActions: []*actionTuple{
				{ActionPromote, newEnabledKey2},
			},
		},
		{
			name: "single_key_old",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
			},
			primary: oldEnabledKey.Name,
			wantActions: []*actionTuple{
				{ActionCreateNew, nil},
			},
		},
		{
			name: "two_enabled_keys_no_action",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				newEnabledKey2,
			},
			primary:     oldEnabledKey.Name,
			wantActions: []*actionTuple{},
		},
		{
			name: "two_enabled_keys_old",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				newEnabledKey,
			},
			primary: newEnabledKey.Name,
			wantActions: []*actionTuple{
				{ActionDisable, oldEnabledKey},
			},
		},
		{
			name: "two_enabled_keys_new",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				newEnabledKey,
			},
			primary: oldEnabledKey.Name,
			wantActions: []*actionTuple{
				{ActionPromote, newEnabledKey},
			},
		},
		{
			name: "three_enabled_keys_no_action",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				newEnabledKey,
				newEnabledKey2,
			},
			primary:     oldEnabledKey.Name,
			wantActions: []*actionTuple{},
		},
		{
			name: "three_enabled_keys_promote",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey2,
				oldEnabledKey,
				newEnabledKey,
			},
			primary: oldEnabledKey2.Name,
			wantActions: []*actionTuple{
				{ActionPromote, newEnabledKey},
			},
		},
		{
			name: "three_enabled_keys",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				newEnabledKey,
				oldEnabledKey2,
			},
			primary: oldEnabledKey.Name,
			wantActions: []*actionTuple{
				{ActionDisable, oldEnabledKey2},
				{ActionPromote, newEnabledKey},
			},
		},
		{
			name: "many_keys",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				newEnabledKey,
				oldDisabledKey,
				newDisabledKey,
				oldDestroyedKey,
			},
			primary: newEnabledKey.Name,
			wantActions: []*actionTuple{
				{ActionDisable, oldEnabledKey},
				{ActionDestroy, oldDisabledKey},
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			output, err := handler.determineActions(ctx, tc.versions, tc.primary, curTime)

			if diff := cmp.Diff(tc.wantActions, output, protocmp.Transform()); diff != "" {
				t.Errorf("Got diff (-want, +got): %v", diff)
			}

			if tc.wantErr != "" {
				if err != nil {
					if diff := cmp.Diff(err.Error(), tc.wantErr); diff != "" {
						t.Errorf("Process got unexpected error substring: %v", diff)
					}
				} else {
					t.Errorf("Expected error, but received nil")
				}
			} else if err != nil {
				t.Errorf("Expected no error, but received \"%v\"", err)
			}
		})
	}
}

func TestPerformActions(t *testing.T) {
	t.Parallel()

	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", "[PROJECT]", "[LOCATION]", "[KEY_RING]", "[CRYPTO_KEY]")
	versionSuffix := "[VERSION]"
	versionName := fmt.Sprintf("%s/cryptoKeyVersions/%s", parent, versionSuffix)

	cases := []struct {
		name             string
		actions          []*actionTuple
		priorPrimary     string
		expectedRequests []proto.Message
		expectedPrimary  string
		wantErr          string
		serverErr        error
	}{
		{
			name: "disable",
			actions: []*actionTuple{
				{
					ActionDisable,
					&kmspb.CryptoKeyVersion{
						State: kmspb.CryptoKeyVersion_ENABLED,
						Name:  versionName,
					},
				},
			},
			wantErr: "",
			expectedRequests: []proto.Message{
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
		},
		{
			name: "create_new_and_promote",
			actions: []*actionTuple{
				{
					ActionCreateNewAndPromote,
					&kmspb.CryptoKeyVersion{
						State: kmspb.CryptoKeyVersion_ENABLED,
						Name:  versionName,
					},
				},
			},
			wantErr: "",
			expectedRequests: []proto.Message{
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName + "-new",
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
			expectedPrimary: PrimaryLabelPrefix + versionSuffix + "-new",
		},
		{
			name: "create_new_and_promote_with_failure",
			actions: []*actionTuple{
				{
					ActionCreateNewAndPromote,
					&kmspb.CryptoKeyVersion{
						State: kmspb.CryptoKeyVersion_ENABLED,
						Name:  versionName,
					},
				},
			},
			serverErr: fmt.Errorf("key creation failed"),
			wantErr:   "1 error occurred:\n\t* key creation failed: rpc error: code = Unknown desc = key creation failed\n\n",
			expectedRequests: []proto.Message{
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
			},
			priorPrimary:    PrimaryLabelPrefix + versionSuffix,
			expectedPrimary: PrimaryLabelPrefix + versionSuffix,
		},
		{
			name: "destroy",
			actions: []*actionTuple{
				{
					ActionDestroy,
					&kmspb.CryptoKeyVersion{
						State: kmspb.CryptoKeyVersion_DISABLED,
						Name:  versionName,
					},
				},
			},
			wantErr: "",
			expectedRequests: []proto.Message{
				&kmspb.DestroyCryptoKeyVersionRequest{
					Name: versionName,
				},
			},
		},
		{
			name: "multi_action",
			actions: []*actionTuple{
				{ActionCreateNew, nil},
				{
					ActionDestroy,
					&kmspb.CryptoKeyVersion{
						Name:  versionName + "2",
						State: kmspb.CryptoKeyVersion_DISABLED,
					},
				},
			},
			priorPrimary:    PrimaryLabelPrefix + versionSuffix,
			expectedPrimary: PrimaryLabelPrefix + versionSuffix,
			wantErr:         "",
			expectedRequests: []proto.Message{
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				&kmspb.GetCryptoKeyVersionRequest{
					Name: versionName + "-new",
				},
				&kmspb.DestroyCryptoKeyVersionRequest{
					Name: versionName + "2",
				},
			},
		},
		{
			name: "test_err",
			actions: []*actionTuple{
				{
					ActionDestroy,
					&kmspb.CryptoKeyVersion{
						Name:  versionName,
						State: kmspb.CryptoKeyVersion_DISABLED,
					},
				},
			},
			serverErr: fmt.Errorf("test error while disabling"),
			wantErr:   "1 error occurred:\n\t* key destroy failed: rpc error: code = Unknown desc = test error while disabling\n\n",
			expectedRequests: []proto.Message{
				&kmspb.DestroyCryptoKeyVersionRequest{
					Name: versionName,
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

			mockKeyManagement := testutil.NewMockKeyManagementServer(parent, versionName, tc.priorPrimary)
			mockKeyManagement.Err = tc.serverErr
			mockKeyManagement.Resps = append(mockKeyManagement.Resps[:0], &kmspb.CryptoKeyVersion{Name: versionName + "-new"})
			serv := grpc.NewServer()
			kmspb.RegisterKeyManagementServiceServer(serv, mockKeyManagement)

			_, conn := pkgtestutil.FakeGRPCServer(t, func(s *grpc.Server) {
				kmspb.RegisterKeyManagementServiceServer(s, mockKeyManagement)
			})

			clientOpt := option.WithGRPCConn(conn)
			t.Cleanup(func() {
				conn.Close()
			})

			c, err := kms.NewKeyManagementClient(ctx, clientOpt)
			if err != nil {
				t.Fatal(err)
			}

			handler := NewRotationHandler(ctx, c, nil)

			gotErr := handler.performActions(ctx, parent, tc.actions)
			if diff := pkgtestutil.DiffErrString(gotErr, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if diff := cmp.Diff(tc.expectedRequests, mockKeyManagement.Reqs, protocmp.Transform()); diff != "" {
				t.Errorf("wrong requests: diff (-want, +got): %s", diff)
			}
			if diff := cmp.Diff(tc.expectedPrimary, mockKeyManagement.Labels["primary"]); diff != "" {
				t.Errorf("wrong primary: diff (-want, +got): %s", diff)
			}
		})
	}
}
