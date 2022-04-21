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

package crypto

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/abcxyz/jvs/pkg/config"
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

type mockKeyManagementServer struct {
	// Embed for forward compatibility.
	// Tests will keep working if more methods are added
	// in the future.
	kmspb.UnimplementedKeyManagementServiceServer

	reqs []proto.Message

	// If set, all calls return this error.
	err error

	// responses to return if err == nil
	resps []proto.Message
}

func (s *mockKeyManagementServer) ListCryptoKeyVersions(ctx context.Context, req *kmspb.ListCryptoKeyVersionsRequest) (*kmspb.ListCryptoKeyVersionsResponse, error) {
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*kmspb.ListCryptoKeyVersionsResponse), nil
}

func (s *mockKeyManagementServer) CreateCryptoKeyVersion(ctx context.Context, req *kmspb.CreateCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*kmspb.CryptoKeyVersion), nil
}

var clientOpt option.ClientOption
var mockKeyManagement = &mockKeyManagementServer{
	UnimplementedKeyManagementServiceServer: kmspb.UnimplementedKeyManagementServiceServer{},
	reqs:                                    make([]proto.Message, 1),
	err:                                     nil,
	resps:                                   make([]proto.Message, 1),
}

func (s *mockKeyManagementServer) DestroyCryptoKeyVersion(ctx context.Context, req *kmspb.DestroyCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*kmspb.CryptoKeyVersion), nil
}

func (s *mockKeyManagementServer) UpdateCryptoKeyVersion(ctx context.Context, req *kmspb.UpdateCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*kmspb.CryptoKeyVersion), nil
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
	propagationTime, err := time.ParseDuration("30m")
	if err != nil {
		t.Error("Couldn't parse propagation time")
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
			KeyTTL:          keyTTL,
			PropagationTime: propagationTime,
			GracePeriod:     gracePeriod,
			DisabledPeriod:  disablePeriod,
		},
		CurrentTime: time.Unix(100*60*60*24, 0), // 100 days after start
	}

	oldEnabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 50 * 60 * 60 * 24}, // 50 days old
		State:      kmspb.CryptoKeyVersion_ENABLED,
	}
	newEnabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 99 * 60 * 60 * 24}, // 2 days old
		State:      kmspb.CryptoKeyVersion_ENABLED,
	}
	newDisabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 90 * 60 * 60 * 24}, // 10 days old
		State:      kmspb.CryptoKeyVersion_DISABLED,
	}
	oldDisabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 1 * 60 * 60 * 24}, // 99 days old
		State:      kmspb.CryptoKeyVersion_DISABLED,
	}
	oldDestroyedKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 1 * 60 * 60 * 24}, // 99 days old
		State:      kmspb.CryptoKeyVersion_DESTROYED,
	}
	pendingGenerationKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 99 * 60 * 60 * 24}, // 1 days old,
		State:      kmspb.CryptoKeyVersion_PENDING_GENERATION,
	}

	tests := []struct {
		name        string
		versions    []*kmspb.CryptoKeyVersion
		wantActions map[*kmspb.CryptoKeyVersion]Action
		wantErr     string
	}{
		{
			name: "single_key_old",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
			},
			wantActions: map[*kmspb.CryptoKeyVersion]Action{
				oldEnabledKey: ActionCreate,
			},
		},
		{
			name: "two_enabled_keys",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				newEnabledKey,
			},
			wantActions: map[*kmspb.CryptoKeyVersion]Action{
				oldEnabledKey: ActionDisable,
				newEnabledKey: ActionNone,
			},
		},
		{
			name: "old_key_and_pending",
			versions: []*kmspb.CryptoKeyVersion{
				oldEnabledKey,
				pendingGenerationKey,
			},
			wantActions: map[*kmspb.CryptoKeyVersion]Action{
				oldEnabledKey:        ActionNone,
				pendingGenerationKey: ActionNone,
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
			output, err := handler.determineActions(tc.versions)

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
		expectedRequests []proto.Message
		wantErr          string
		serverErr        error
	}{
		{
			name: "disable",
			actions: map[*kmspb.CryptoKeyVersion]Action{
				{
					State: kmspb.CryptoKeyVersion_ENABLED,
				}: ActionDisable,
			},
			wantErr: "",
			expectedRequests: []proto.Message{
				&kmspb.UpdateCryptoKeyVersionRequest{
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{
						State: kmspb.CryptoKeyVersion_DISABLED,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"state"},
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
				}: ActionCreate,
			},
			wantErr: "",
			expectedRequests: []proto.Message{
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
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
					State: kmspb.CryptoKeyVersion_ENABLED,
				}: ActionDisable,
				{
					Name:  versionName,
					State: kmspb.CryptoKeyVersion_ENABLED,
				}: ActionCreate,
				{
					Name:  versionName,
					State: kmspb.CryptoKeyVersion_DISABLED,
				}: ActionDestroy,
			},
			wantErr: "",
			expectedRequests: []proto.Message{
				&kmspb.UpdateCryptoKeyVersionRequest{
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{
						State: kmspb.CryptoKeyVersion_DISABLED,
					},
					UpdateMask: &fieldmaskpb.FieldMask{
						Paths: []string{"state"},
					},
				},
				&kmspb.CreateCryptoKeyVersionRequest{
					Parent:           parent,
					CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
				},
				&kmspb.DestroyCryptoKeyVersionRequest{
					Name: versionName,
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
			mockKeyManagement.err = nil
			mockKeyManagement.reqs = nil
			mockKeyManagement.err = tc.serverErr

			mockKeyManagement.resps = append(mockKeyManagement.resps[:0], &kmspb.CryptoKeyVersion{})

			gotErr := handler.performActions(ctx, tc.actions)

			if err != nil {
				t.Fatal(err)
			}

			if want, got := tc.expectedRequests, mockKeyManagement.reqs; !slicesEq(want, got) {
				t.Errorf("wrong request %q, want %q", got, want)
			}
			errCmp(t, tc.wantErr, gotErr)
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

func errCmp(t *testing.T, wantErr string, gotErr error) {
	if wantErr != "" {
		if gotErr != nil {
			if diff := cmp.Diff(gotErr.Error(), wantErr); diff != "" {
				t.Errorf("Process got unexpected error substring: %v", diff)
			}
		} else {
			t.Errorf("Expected error, but received nil")
		}
	} else if gotErr != nil {
		t.Errorf("Expected no error, but received \"%v\"", gotErr)
	}
}
