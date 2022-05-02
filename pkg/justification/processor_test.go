package justification

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	v0 "github.com/abcxyz/jvs/api/v0"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	crypto2 "github.com/abcxyz/jvs/pkg/crypto"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/golang-jwt/jwt"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
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

	privateKey *ecdsa.PrivateKey
	publicKey  string
}

func (s *mockKeyManagementServer) ListCryptoKeyVersions(ctx context.Context, req *kmspb.ListCryptoKeyVersionsRequest) (*kmspb.ListCryptoKeyVersionsResponse, error) {
	s.reqs = append(s.reqs, req)
	if s.err != nil {
		return nil, s.err
	}
	return &kmspb.ListCryptoKeyVersionsResponse{
		CryptoKeyVersions: []*kmspb.CryptoKeyVersion{
			{
				Name:  "testkey",
				State: kmspb.CryptoKeyVersion_ENABLED,
			},
		},
	}, nil
}

func (s *mockKeyManagementServer) AsymmetricSign(ctx context.Context, req *kmspb.AsymmetricSignRequest) (*kmspb.AsymmetricSignResponse, error) {
	sig, err := ecdsa.SignASN1(rand.Reader, s.privateKey, req.Digest.GetSha256())
	if err != nil {
		return nil, s.err
	}
	return &kmspb.AsymmetricSignResponse{
		Signature: sig,
	}, nil
}

func (s *mockKeyManagementServer) GetPublicKey(ctx context.Context, req *kmspb.GetPublicKeyRequest) (*kmspb.PublicKey, error) {
	return &kmspb.PublicKey{
		Pem: s.publicKey,
	}, nil
}

var clientOpt option.ClientOption
var mockKeyManagement = &mockKeyManagementServer{
	UnimplementedKeyManagementServiceServer: kmspb.UnimplementedKeyManagementServiceServer{},
	reqs:                                    make([]proto.Message, 1),
	err:                                     nil,
	resps:                                   make([]proto.Message, 1),
}

func TestCreateToken(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	mockKeyManagement.privateKey = privateKey
	x509EncodedPub, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		t.Fatal(err)
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})
	mockKeyManagement.publicKey = string(pemEncodedPub)

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

	signer := &crypto2.KMSSigner{
		Config:    &config.JustificationConfig{},
		KMSClient: c,
	}
	processor := &Processor{
		Signer: signer,
	}
	hour, err := time.ParseDuration("3600s")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		request   *jvspb.CreateJustificationRequest
		wantErr   string
		serverErr error
	}{
		{
			name: "happy_path",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "test",
					},
				},
				Ttl: durationpb.New(hour),
			},
		},
		{
			name: "no_justification",
			request: &jvspb.CreateJustificationRequest{
				Ttl: durationpb.New(hour),
			},
			wantErr: "couldn't validate request",
		},
		{
			name: "no_ttl",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "test",
					},
				},
			},
			wantErr: "couldn't validate request",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockKeyManagement.err = nil
			mockKeyManagement.reqs = nil
			mockKeyManagement.err = tc.serverErr

			mockKeyManagement.resps = append(mockKeyManagement.resps[:0], &kmspb.CryptoKeyVersion{})

			response, gotErr := processor.CreateToken(ctx, tc.request)
			testutil.ErrCmp(t, tc.wantErr, gotErr)

			if gotErr == nil {
				if err := signer.VerifyJWTString(ctx, response); err != nil {
					t.Errorf("Unable to verify signed jwt. %v", err)
				}

				claims := &v0.JVSClaims{}
				_, err := jwt.ParseWithClaims(response, claims, func(token *jwt.Token) (interface{}, error) {
					return privateKey.Public(), nil
				})
				if err != nil {
					t.Errorf("Unable to parse created jwt string. %v", err)
				}
				validateClaims(t, claims, tc.request.Justifications)
			}
		})
	}
}

func validateClaims(t testing.TB, provided *v0.JVSClaims, expectedJustifications []*jvspb.Justification) {
	// test the standard claims filled by processor
	var err *multierror.Error
	if provided.StandardClaims.Issuer != jvsIssuer {
		err = multierror.Append(err, fmt.Errorf("audience value %s incorrect, expected %s", provided.StandardClaims.Issuer, jvsIssuer))
	}
	// TODO: as we add more standard claims, add more validations.

	if err.ErrorOrNil() != nil {
		t.Errorf("standard claims weren't set correctly. %v", err)
	}

	if len(provided.Justifications) != len(expectedJustifications) {
		t.Errorf("Number of justifications was incorrect.\n got: %v\n want: %v", provided.Justifications, expectedJustifications)
	}

	for _, j := range provided.Justifications {
		found := false
		for i, expectedJ := range expectedJustifications {
			if j.Value == expectedJ.Value && j.Category == expectedJ.Category {
				expectedJustifications = append(expectedJustifications[:i], expectedJustifications[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Justifications didn't match.\n got: %v\n want: %v", provided.Justifications, expectedJustifications)
		}
	}
}
