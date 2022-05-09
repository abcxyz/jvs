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
	"log"
	"net"
	"testing"
	"time"

	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/testutil"
	"google.golang.org/protobuf/types/known/timestamppb"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

var clientOpt option.ClientOption
var mockKeyManagement = &testutil.MockKeyManagementServer{
	UnimplementedKeyManagementServiceServer: kmspb.UnimplementedKeyManagementServiceServer{},
	Reqs:                                    make([]proto.Message, 1),
	Err:                                     nil,
	Resps:                                   make([]proto.Message, 1),
}

func TestGetKeyNameFromVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
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

	for _, tc := range tests {
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

	handler := &RotationHandler{
		KmsClient: nil,
		CryptoConfig: &config.CryptoConfig{
			KeyTTL:         keyTTL,
			GracePeriod:    gracePeriod,
			DisabledPeriod: disablePeriod,
		},
		CurrentTime: time.Unix(100*60*60*24, 0), // 100 days after start
	}

	oldEnabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 50 * 60 * 60 * 24}, // 50 days old
		State:      kmspb.CryptoKeyVersion_ENABLED,
		Name:       "oldEnabledKey",
	}
	oldEnabledKey2 := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 50 * 60 * 60 * 24}, // 50 days old
		State:      kmspb.CryptoKeyVersion_ENABLED,
		Name:       "oldEnabledKey2",
	}
	newEnabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 99 * 60 * 60 * 24}, // 2 days old
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

	tests := []struct {
		name         string
		versions     []*kmspb.CryptoKeyVersion
		activeStates map[string]VersionState
		wantActions  map[*kmspb.CryptoKeyVersion]Action
		wantErr      string
	}{
		{
			name:         "no_key",
			versions:     []*kmspb.CryptoKeyVersion{},
			activeStates: map[string]VersionState{},
			wantActions: map[*kmspb.CryptoKeyVersion]Action{
				nil: ActionCreateNewAndPromote,
			},
		},
		{
			name: "single_key_old",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
			},
			activeStates: map[string]VersionState{
				oldEnabledKey.Name: VersionStatePrimary,
			},
			wantActions: map[*kmspb.CryptoKeyVersion]Action{
				oldEnabledKey: ActionCreateNew,
			},
		},
		{
			name: "two_enabled_keys_old",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				newEnabledKey,
			},
			activeStates: map[string]VersionState{
				newEnabledKey.Name: VersionStatePrimary,
				oldEnabledKey.Name: VersionStateOld,
			},
			wantActions: map[*kmspb.CryptoKeyVersion]Action{
				oldEnabledKey: ActionDisable,
				newEnabledKey: ActionNone,
			},
		},
		{
			// this is intended to replicate the state if we had rolled back a key. we expect the old key to be primary, and the newer
			// key to be marked as "old" in the database.
			name: "two_enabled_keys_rollback",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				newEnabledKey,
			},
			activeStates: map[string]VersionState{
				newEnabledKey.Name: VersionStateOld,
				oldEnabledKey.Name: VersionStatePrimary,
			},
			wantActions: map[*kmspb.CryptoKeyVersion]Action{
				oldEnabledKey: ActionCreateNew,
				newEnabledKey: ActionNone,
			},
		},
		{
			name: "two_enabled_keys_new",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				newEnabledKey,
			},
			activeStates: map[string]VersionState{
				newEnabledKey.Name: VersionStateNew,
				oldEnabledKey.Name: VersionStatePrimary,
			},
			wantActions: map[*kmspb.CryptoKeyVersion]Action{
				oldEnabledKey: ActionDemote,
				newEnabledKey: ActionPromote,
			},
		},
		{
			name: "three_enabled_keys",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				newEnabledKey,
				oldEnabledKey2,
			},
			activeStates: map[string]VersionState{
				newEnabledKey.Name:  VersionStateNew,
				oldEnabledKey.Name:  VersionStatePrimary,
				oldEnabledKey2.Name: VersionStateOld,
			},
			wantActions: map[*kmspb.CryptoKeyVersion]Action{
				oldEnabledKey:  ActionDemote,
				newEnabledKey:  ActionPromote,
				oldEnabledKey2: ActionDisable,
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
			activeStates: map[string]VersionState{
				newEnabledKey.Name: VersionStatePrimary,
				oldEnabledKey.Name: VersionStateOld,
			},
			wantActions: map[*kmspb.CryptoKeyVersion]Action{
				oldEnabledKey:   ActionDisable,
				newEnabledKey:   ActionNone,
				oldDisabledKey:  ActionDestroy,
				newDisabledKey:  ActionNone,
				oldDestroyedKey: ActionNone,
			},
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			output, err := handler.determineActions(tc.versions, tc.activeStates)

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
	ctx := context.Background()

	serv := grpc.NewServer()
	kmspb.RegisterKeyManagementServiceServer(serv, mockKeyManagement)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatal(err)
	}
	go serv.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	clientOpt = option.WithGRPCConn(conn)
	t.Cleanup(func() {
		conn.Close()
	})

	c, err := kms.NewKeyManagementClient(ctx, clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	handler := &RotationHandler{
		KmsClient:    c,
		CryptoConfig: &config.CryptoConfig{},
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", "[PROJECT]", "[LOCATION]", "[KEY_RING]", "[CRYPTO_KEY]")
	versionName := fmt.Sprintf("%s/cryptoKeyVersions/%s", parent, "[VERSION]")

	tests := []struct {
		name             string
		actions          map[*kmspb.CryptoKeyVersion]Action
		priorStates      map[string]VersionState
		expectedRequests []proto.Message
		expectedStates   map[string]VersionState
		wantErr          string
		serverErr        error
	}{
		{
			name: "disable",
			actions: map[*kmspb.CryptoKeyVersion]Action{
				{
					State: kmspb.CryptoKeyVersion_ENABLED,
					Name:  versionName,
				}: ActionDisable,
			},
			priorStates: map[string]VersionState{
				versionName: VersionStatePrimary, // should be in db before, but not after.
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
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Labels: nil,
						Name:   parent,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				},
			},
		},
		{
			name: "create",
			actions: map[*kmspb.CryptoKeyVersion]Action{
				{
					Name:  versionName,
					State: kmspb.CryptoKeyVersion_ENABLED,
				}: ActionCreateNew,
			},
			wantErr: "",
			expectedRequests: []proto.Message{
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Labels: map[string]string{versionName + "-new": VersionStateNew.String()},
						Name:   parent,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				},
			},
			expectedStates: map[string]VersionState{
				versionName + "-new": VersionStateNew,
			},
		},
		{
			name: "destroy",
			actions: map[*kmspb.CryptoKeyVersion]Action{
				{
					Name:  versionName,
					State: kmspb.CryptoKeyVersion_DISABLED,
				}: ActionDestroy,
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
			actions: map[*kmspb.CryptoKeyVersion]Action{
				{
					Name:  versionName,
					State: kmspb.CryptoKeyVersion_ENABLED,
				}: ActionCreateNew,
				{
					Name:  versionName + "2",
					State: kmspb.CryptoKeyVersion_DISABLED,
				}: ActionDestroy,
			},
			priorStates: map[string]VersionState{
				versionName: VersionStatePrimary,
			},
			expectedStates: map[string]VersionState{
				versionName:          VersionStatePrimary,
				versionName + "-new": VersionStateNew,
			},
			wantErr: "",
			expectedRequests: []proto.Message{
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				&kmspb.GetCryptoKeyRequest{
					Name: parent,
				},
				&kmspb.UpdateCryptoKeyRequest{
					CryptoKey: &kmspb.CryptoKey{
						Labels: map[string]string{
							versionName + "-new": VersionStateNew.String(),
							versionName:          VersionStatePrimary.String(),
						},
						Name: parent,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"labels"},
					},
				},
				&kmspb.DestroyCryptoKeyVersionRequest{
					Name: versionName + "2",
				},
			},
		},
		{
			name: "test_err",
			actions: map[*kmspb.CryptoKeyVersion]Action{
				{
					Name:  versionName,
					State: kmspb.CryptoKeyVersion_DISABLED,
				}: ActionDestroy,
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

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockKeyManagement.Reqs = nil
			mockKeyManagement.Err = tc.serverErr
			mockKeyManagement.KeyName = parent
			mockKeyManagement.VersionName = versionName

			mockKeyManagement.Resps = append(mockKeyManagement.Resps[:0], &kmspb.CryptoKeyVersion{Name: versionName + "-new"})

			mockKeyManagement.Labels = make(map[string]string)
			for key, state := range tc.priorStates {
				mockKeyManagement.Labels[key] = state.String()
			}

			gotErr := handler.performActions(ctx, parent, tc.actions)

			if err != nil {
				t.Fatal(err)
			}

			if want, got := tc.expectedRequests, mockKeyManagement.Reqs; !slicesEq(want, got) {
				for _, msg := range got {
					t.Errorf("gotty: %s", msg)
				}
				for _, msg := range want {
					t.Errorf("wanty: %s", msg)
				}
				t.Errorf("wrong requests %v, want %v", got, want)
			}
			testutil.ErrCmp(t, tc.wantErr, gotErr)

			got := make(map[string]VersionState)
			for key, value := range mockKeyManagement.Labels {
				got[key] = GetVersionState(value)
			}

			if len(got) == 0 {
				got = nil
			}
			if diff := cmp.Diff(tc.expectedStates, got); diff != "" {
				t.Errorf("Got diff (-want, +got): %v", diff)
			}

		})
	}
}

func slicesEq(a, b []proto.Message) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		found := false
		for j := range b {
			if proto.Equal(a[i], b[j]) {
				found = true
				b = append(b[:j], b[j+1:]...) // remove from slice
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
