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
	"os"
	"strings"
	"testing"

	"google-on-gcp/jvs/services/go/apis/v1alpha1"

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
	kmspb.KeyManagementServiceServer

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
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return s.resps[0].(*kmspb.CryptoKeyVersion), nil
}

var clientOpt option.ClientOption
var mockKeyManagement = &mockKeyManagementServer{
	KeyManagementServiceServer: nil,
	reqs:                       make([]proto.Message, 1),
	err:                        nil,
	resps:                      make([]proto.Message, 1),
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

// Set up mock server
func TestMain(m *testing.M) {
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

	os.Exit(m.Run())
}

func TestDestroy(t *testing.T) {
	ctx := context.Background()
	var name2 string = "name2-1052831874"
	var importJob string = "importJob2125587491"
	var importFailureReason string = "importFailureReason-494073229"
	var expectedResponse = &kmspb.CryptoKeyVersion{
		Name:                name2,
		ImportJob:           importJob,
		ImportFailureReason: importFailureReason,
	}

	mockKeyManagement.err = nil
	mockKeyManagement.reqs = nil

	mockKeyManagement.resps = append(mockKeyManagement.resps[:0], expectedResponse)

	var formattedName string = fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s/cryptoKeyVersions/%s", "[PROJECT]", "[LOCATION]", "[KEY_RING]", "[CRYPTO_KEY]", "[CRYPTO_KEY_VERSION]")
	var request = &kmspb.DestroyCryptoKeyVersionRequest{
		Name: formattedName,
	}

	c, err := kms.NewKeyManagementClient(ctx, clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	handler := &RotationHandler{
		KmsClient: c,
		JvsConfig: v1alpha1.Config{},
		KeyName:   "",
	}

	err = handler.performDestroy(ctx, &kmspb.CryptoKeyVersion{Name: formattedName})

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockKeyManagement.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	var name string = "name3373707"
	var importJob string = "importJob2125587491"
	var importFailureReason string = "importFailureReason-494073229"
	var expectedResponse = &kmspb.CryptoKeyVersion{
		Name:                name,
		ImportJob:           importJob,
		ImportFailureReason: importFailureReason,
	}

	mockKeyManagement.err = nil
	mockKeyManagement.reqs = nil

	mockKeyManagement.resps = append(mockKeyManagement.resps[:0], expectedResponse)

	var formattedParent string = fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", "[PROJECT]", "[LOCATION]", "[KEY_RING]", "[CRYPTO_KEY]")
	var cryptoKeyVersion *kmspb.CryptoKeyVersion = &kmspb.CryptoKeyVersion{
		Name: fmt.Sprintf("%s/cryptoKeyVersions/%s", formattedParent, "[VERSION]"),
	}
	var request = &kmspb.CreateCryptoKeyVersionRequest{
		Parent:           formattedParent,
		CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
	}

	c, err := kms.NewKeyManagementClient(ctx, clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	handler := &RotationHandler{
		KmsClient: c,
		JvsConfig: v1alpha1.Config{},
		KeyName:   "",
	}

	err = handler.performCreate(ctx, cryptoKeyVersion)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockKeyManagement.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}
}

func TestDisable(t *testing.T) {
	ctx := context.Background()
	var name string = "name3373707"
	var importJob string = "importJob2125587491"
	var importFailureReason string = "importFailureReason-494073229"
	var expectedResponse = &kmspb.CryptoKeyVersion{
		Name:                name,
		ImportJob:           importJob,
		ImportFailureReason: importFailureReason,
	}

	mockKeyManagement.err = nil
	mockKeyManagement.reqs = nil

	mockKeyManagement.resps = append(mockKeyManagement.resps[:0], expectedResponse)

	var cryptoKeyVersion *kmspb.CryptoKeyVersion = &kmspb.CryptoKeyVersion{
		State: kmspb.CryptoKeyVersion_DISABLED,
	}
	var updateMask *fieldmaskpb.FieldMask = &fieldmaskpb.FieldMask{
		Paths: []string{"state"},
	}
	var request = &kmspb.UpdateCryptoKeyVersionRequest{
		CryptoKeyVersion: cryptoKeyVersion,
		UpdateMask:       updateMask,
	}

	c, err := kms.NewKeyManagementClient(ctx, clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	handler := &RotationHandler{
		KmsClient: c,
		JvsConfig: v1alpha1.Config{},
		KeyName:   "",
	}

	err = handler.performDisable(ctx, cryptoKeyVersion)

	if err != nil {
		t.Fatal(err)
	}

	if want, got := request, mockKeyManagement.reqs[0]; !proto.Equal(want, got) {
		t.Errorf("wrong request %q, want %q", got, want)
	}
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
