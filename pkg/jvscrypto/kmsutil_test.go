package jvscrypto

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/sethvargo/go-gcpkms/pkg/gcpkms"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
)

func TestVerifyJWTString(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var clientOpt option.ClientOption
	mockKMS := &testutil.MockKeyManagementServer{
		UnimplementedKeyManagementServiceServer: kmspb.UnimplementedKeyManagementServiceServer{},
		Reqs:                                    make([]proto.Message, 1),
		Err:                                     nil,
		Resps:                                   make([]proto.Message, 1),
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	mockKMS.PrivateKey = privateKey
	x509EncodedPub, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		t.Fatal(err)
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})
	mockKMS.PublicKey = string(pemEncodedPub)

	serv := grpc.NewServer()
	kmspb.RegisterKeyManagementServiceServer(serv, mockKMS)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	// not checked, but makes linter happy
	errs := make(chan error, 1)
	go func() {
		errs <- serv.Serve(lis)
		close(errs)
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	clientOpt = option.WithGRPCConn(conn)
	t.Cleanup(func() {
		conn.Close()
	})

	kms, err := kms.NewKeyManagementClient(ctx, clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	parent := "test-key"
	signer, err := gcpkms.NewSigner(ctx, kms, parent)
	if err != nil {
		t.Fatal(err)
	}

	claims := &jwt.StandardClaims{
		Audience:  "test_aud",
		ExpiresAt: 100,
		Id:        uuid.New().String(),
		IssuedAt:  10,
		Issuer:    "test_iss",
		NotBefore: 10,
		Subject:   "test_sub",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	validJWT, err := SignToken(token, signer)
	if err != nil {
		t.Fatal("Couldn't sign token.")
	}

	unsignedJWT, err := token.SigningString()
	if err != nil {
		t.Fatal("Couldn't get signing string.")
	}

	invalidSignatureJWT := unsignedJWT + ".SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c" // signature from a different JWT

	tests := []struct {
		name    string
		jwt     string
		wantErr string
	}{
		{
			name: "happy_path",
			jwt:  validJWT,
		},
		{
			name:    "unsigned",
			jwt:     unsignedJWT,
			wantErr: "invalid jwt string",
		},
		{
			name:    "invalid",
			jwt:     invalidSignatureJWT,
			wantErr: "unable to verify signed jwt string. crypto/ecdsa: verification error",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := VerifyJWTString(ctx, kms, "projects/*/locations/location1/keyRings/keyring1/cryptoKeys/key1", tc.jwt)
			testutil.ErrCmp(t, tc.wantErr, err)
		})
	}
}
