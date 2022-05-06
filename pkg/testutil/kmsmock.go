package testutil

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"

	"github.com/golang/protobuf/proto"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

type MockKeyManagementServer struct {
	// Embed for forward compatibility.
	// Tests will keep working if more methods are added
	// in the future.
	kmspb.UnimplementedKeyManagementServiceServer

	Reqs []proto.Message

	// If set, all calls return this error.
	Err error

	// responses to return if err == nil
	Resps []proto.Message

	PrivateKey *ecdsa.PrivateKey
	PublicKey  string
}

const TestKeyName = "projects/proj1/locations/loc1/keyRings/kr1/cryptoKeys/key1"

func (s *MockKeyManagementServer) CreateCryptoKeyVersion(ctx context.Context, req *kmspb.CreateCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	s.Reqs = append(s.Reqs, req)
	if s.Err != nil {
		return nil, s.Err
	}
	return s.Resps[0].(*kmspb.CryptoKeyVersion), nil
}

func (s *MockKeyManagementServer) ListCryptoKeyVersions(ctx context.Context, req *kmspb.ListCryptoKeyVersionsRequest) (*kmspb.ListCryptoKeyVersionsResponse, error) {
	s.Reqs = append(s.Reqs, req)
	if s.Err != nil {
		return nil, s.Err
	}
	return &kmspb.ListCryptoKeyVersionsResponse{
		CryptoKeyVersions: []*kmspb.CryptoKeyVersion{
			{
				Name:  TestKeyName,
				State: kmspb.CryptoKeyVersion_ENABLED,
			},
		},
	}, nil
}

func (s *MockKeyManagementServer) GetCryptoKey(ctx context.Context, req *kmspb.GetCryptoKeyRequest) (*kmspb.CryptoKey, error) {
	s.Reqs = append(s.Reqs, req)
	if s.Err != nil {
		return nil, s.Err
	}
	return &kmspb.CryptoKey{
		Primary: &kmspb.CryptoKeyVersion{
			Name:  TestKeyName,
			State: kmspb.CryptoKeyVersion_ENABLED,
		},
	}, nil
}

func (s *MockKeyManagementServer) AsymmetricSign(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
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
	s.Reqs = append(s.Reqs, req)
	if s.Err != nil {
		return nil, s.Err
	}
	return s.Resps[0].(*kmspb.CryptoKeyVersion), nil
}

func (s *MockKeyManagementServer) UpdateCryptoKeyVersion(ctx context.Context, req *kmspb.UpdateCryptoKeyVersionRequest) (*kmspb.CryptoKeyVersion, error) {
	s.Reqs = append(s.Reqs, req)
	if s.Err != nil {
		return nil, s.Err
	}
	return s.Resps[0].(*kmspb.CryptoKeyVersion), nil
}
