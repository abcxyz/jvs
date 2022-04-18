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
	"strings"
	"testing"
	"time"

	"github.com/abcxyz/jvs/apis/v1alpha1"

	"google.golang.org/protobuf/types/known/timestamppb"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*kmspb.ListCryptoKeyVersionsResponse), nil
}

func (s *mockKeyManagementServer) CreateCryptoKeyVersion(ctx context.Context, req *kmspb.CreateCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	log.Print("adding create request")
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
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*kmspb.CryptoKeyVersion), nil
}

func (s *mockKeyManagementServer) UpdateCryptoKeyVersion(ctx context.Context, req *kmspb.UpdateCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	if xg := md["x-goog-api-client"]; len(xg) == 0 || !strings.Contains(xg[0], "gl-go/") {
		return nil, fmt.Errorf("x-goog-api-client = %v, expected gl-go key", xg)
	}
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
		CryptoConfig: &v1alpha1.CryptoConfig{
			KeyTTL:          keyTTL,
			PropagationTime: propagationTime,
			GracePeriod:     gracePeriod,
			DisabledPeriod:  disablePeriod,
		},
		KeyName:     "projects/project_1/locations/location_1/keyRings/keyring_1/cryptoKeys/key_1",
		CurrentTime: time.Unix(100*60*60*24, 0), // 100 days after start
	}

	oldEnabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 50 * 60 * 60 * 24}, // 50 days after start,
		State:      kmspb.CryptoKeyVersion_ENABLED,
	}
	newEnabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 99 * 60 * 60 * 24}, // 1 days after start,
		State:      kmspb.CryptoKeyVersion_ENABLED,
	}
	newDisabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 90 * 60 * 60 * 24}, // 10 days after start,
		State:      kmspb.CryptoKeyVersion_DISABLED,
	}
	oldDisabledKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 1 * 60 * 60 * 24}, // 99 days after start,
		State:      kmspb.CryptoKeyVersion_DISABLED,
	}
	oldDestroyedKey := &kmspb.CryptoKeyVersion{
		CreateTime: &timestamppb.Timestamp{Seconds: 1 * 60 * 60 * 24}, // 99 days after start,
		State:      kmspb.CryptoKeyVersion_DESTROYED,
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
	//t.Parallel()
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
		CryptoConfig: &v1alpha1.CryptoConfig{},
		KeyName:      "",
	}

	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", "[PROJECT]", "[LOCATION]", "[KEY_RING]", "[CRYPTO_KEY]")
	versionName := fmt.Sprintf("%s/cryptoKeyVersions/%s", parent, "[VERSION]")

	tests := []struct {
		name             string
		cryptoKeyVersion *kmspb.CryptoKeyVersion
		action           Action
		expectedRequest  proto.Message
		wantErr          string
	}{
		{
			name: "disable",
			cryptoKeyVersion: &kmspb.CryptoKeyVersion{
				State: kmspb.CryptoKeyVersion_DISABLED,
			},
			action:  ActionDisable,
			wantErr: "",
			expectedRequest: &kmspb.UpdateCryptoKeyVersionRequest{
				CryptoKeyVersion: &kmspb.CryptoKeyVersion{
					State: kmspb.CryptoKeyVersion_DISABLED,
				},
				UpdateMask: &fieldmaskpb.FieldMask{
					Paths: []string{"state"},
				},
			},
		},
		{
			name: "create",
			cryptoKeyVersion: &kmspb.CryptoKeyVersion{
				Name:  versionName,
				State: kmspb.CryptoKeyVersion_ENABLED,
			},
			action:  ActionCreate,
			wantErr: "",
			expectedRequest: &kmspb.CreateCryptoKeyVersionRequest{
				Parent:           parent,
				CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
			},
		},
		{
			name: "destroy",
			cryptoKeyVersion: &kmspb.CryptoKeyVersion{
				Name:  versionName,
				State: kmspb.CryptoKeyVersion_DISABLED,
			},
			action:  ActionDestroy,
			wantErr: "",
			expectedRequest: &kmspb.DestroyCryptoKeyVersionRequest{
				Name: versionName,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockKeyManagement.err = nil
			mockKeyManagement.reqs = nil

			mockKeyManagement.resps = append(mockKeyManagement.resps[:0], &kmspb.CryptoKeyVersion{})

			actions := make(map[*kmspb.CryptoKeyVersion]Action)
			actions[tc.cryptoKeyVersion] = tc.action

			gotErr := handler.performActions(ctx, actions)

			if err != nil {
				t.Fatal(err)
			}

			if want, got := tc.expectedRequest, mockKeyManagement.reqs[0]; !proto.Equal(want, got) {
				t.Errorf("wrong request %q, want %q", got, want)
			}
			errCmp(t, tc.wantErr, gotErr)
		})
	}
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
