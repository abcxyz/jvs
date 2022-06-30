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

// Package testutil provides utilities that are intended to enable easier
// and more concise writing of unit test code.
package testutil

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"sync"

	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/protobuf/proto"
)

type MockKeyManagementServer struct {
	// Embed for forward compatibility.
	// Tests will keep working if more methods are added
	// in the future.
	kmspb.UnimplementedKeyManagementServiceServer

	reqMu sync.Mutex
	Reqs  []proto.Message

	// If set, all calls return this error.
	Err error

	// responses to return if err == nil
	Resps []proto.Message

	Labels map[string]string

	PrivateKey  *ecdsa.PrivateKey
	PublicKey   string
	KeyName     string
	VersionName string
	NumVersions int
}

func (s *MockKeyManagementServer) CreateCryptoKeyVersion(ctx context.Context, req *kmspb.CreateCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	s.reqMu.Lock()
	defer s.reqMu.Unlock()
	s.Reqs = append(s.Reqs, req)
	if s.Err != nil {
		return nil, s.Err
	}
	return firstAsCryptoKeyVersion(s.Resps[0])
}

func (s *MockKeyManagementServer) ListCryptoKeyVersions(ctx context.Context, req *kmspb.ListCryptoKeyVersionsRequest) (*kmspb.ListCryptoKeyVersionsResponse, error) {
	s.reqMu.Lock()
	defer s.reqMu.Unlock()
	s.Reqs = append(s.Reqs, req)
	if s.Err != nil {
		return nil, s.Err
	}
	list := make([]*kmspb.CryptoKeyVersion, 0)
	for i := 0; i < s.NumVersions; i++ {
		list = append(list, &kmspb.CryptoKeyVersion{
			Name:  fmt.Sprintf("%s-%d", s.VersionName, i),
			State: kmspb.CryptoKeyVersion_ENABLED,
		})
	}
	return &kmspb.ListCryptoKeyVersionsResponse{
		CryptoKeyVersions: list,
	}, nil
}

func (s *MockKeyManagementServer) GetCryptoKey(ctx context.Context, req *kmspb.GetCryptoKeyRequest) (*kmspb.CryptoKey, error) {
	s.reqMu.Lock()
	defer s.reqMu.Unlock()
	s.Reqs = append(s.Reqs, req)
	if s.Err != nil {
		return nil, s.Err
	}
	return &kmspb.CryptoKey{
		Name:   s.KeyName,
		Labels: s.Labels,
	}, nil
}

func (s *MockKeyManagementServer) GetCryptoKeyVersion(ctx context.Context, req *kmspb.GetCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	s.reqMu.Lock()
	defer s.reqMu.Unlock()
	s.Reqs = append(s.Reqs, req)
	if s.Err != nil {
		return nil, s.Err
	}
	return &kmspb.CryptoKeyVersion{
		Name: req.Name,
	}, nil
}

func (s *MockKeyManagementServer) AsymmetricSign(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
	s.reqMu.Lock()
	defer s.reqMu.Unlock()
	s.Reqs = append(s.Reqs, req)
	sig, err := ecdsa.SignASN1(rand.Reader, s.PrivateKey, req.Digest.GetSha256())
	if err != nil {
		return nil, s.Err
	}
	return &kmspb.AsymmetricSignResponse{
		Signature: sig,
	}, nil
}

func (s *MockKeyManagementServer) GetPublicKey(ctx context.Context, req *kmspb.GetPublicKeyRequest) (*kmspb.PublicKey, error) {
	return &kmspb.PublicKey{
		Pem:       s.PublicKey,
		Algorithm: kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256,
	}, nil
}

func (s *MockKeyManagementServer) DestroyCryptoKeyVersion(ctx context.Context, req *kmspb.DestroyCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	s.reqMu.Lock()
	defer s.reqMu.Unlock()
	s.Reqs = append(s.Reqs, req)
	if s.Err != nil {
		return nil, s.Err
	}
	return firstAsCryptoKeyVersion(s.Resps[0])
}

func (s *MockKeyManagementServer) UpdateCryptoKeyVersion(ctx context.Context, req *kmspb.UpdateCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	s.reqMu.Lock()
	defer s.reqMu.Unlock()
	s.Reqs = append(s.Reqs, req)
	if s.Err != nil {
		return nil, s.Err
	}
	return firstAsCryptoKeyVersion(s.Resps[0])
}

func firstAsCryptoKeyVersion(m proto.Message) (*kmspb.CryptoKeyVersion, error) {
	ver, ok := m.(*kmspb.CryptoKeyVersion)
	if !ok {
		return nil, fmt.Errorf("response is not a *kmspb.CryptoKeyVersion (%T)", m)
	}
	return ver, nil
}

func (s *MockKeyManagementServer) UpdateCryptoKey(ctx context.Context, req *kmspb.UpdateCryptoKeyRequest) (*kmspb.CryptoKey, error) {
	s.Reqs = append(s.Reqs, req)
	s.Labels = req.CryptoKey.Labels

	return &kmspb.CryptoKey{}, nil
}
