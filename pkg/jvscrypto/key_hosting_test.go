package jvscrypto

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net"
	"testing"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/grpc"
)

func TestJWKSetFormattedString(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var clientOpt option.ClientOption
	var mockKMSServer = &testutil.MockKeyManagementServer{
		UnimplementedKeyManagementServiceServer: kmspb.UnimplementedKeyManagementServiceServer{},
		Reqs:                                    make([]proto.Message, 1),
		Err:                                     nil,
		Resps:                                   make([]proto.Message, 1),
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	mockKMSServer.PrivateKey = privateKey
	x509EncodedPub, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		t.Fatal(err)
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})
	mockKMSServer.PublicKey = string(pemEncodedPub)

	serv := grpc.NewServer()
	kmspb.RegisterKeyManagementServiceServer(serv, mockKMSServer)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	go serv.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
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

	ks := &KeyServer{
		KmsClient:    kms,
		CryptoConfig: &config.CryptoConfig{},
		StateStore:   &KeyLabelStateStore{KMSClient: kms},
	}

	key := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", "[PROJECT]", "[LOCATION]", "[KEY_RING]", "[CRYPTO_KEY]")
	versionSuffix := "[VERSION]"

	tests := []struct {
		name         string
		storageState map[string]string
		wantOutput   string
		wantErr      string
	}{
		{
			name: "happy-path",
			storageState: map[string]string{
				"ver_" + versionSuffix: VersionStatePrimary.String(),
			},
			wantOutput: fmt.Sprintf("{\"keys\":[{\"crv\":\"P-256\",\"kid\":\"ver_[VERSION]\",\"kty\":\"EC\",\"x\":\"%s\",\"y\":\"%s\"}]}",
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.X.Bytes()),
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.Y.Bytes())),
		},
		{
			name: "multi-key",
			storageState: map[string]string{
				"ver_" + versionSuffix:       VersionStatePrimary.String(),
				"ver_" + versionSuffix + "2": VersionStateNew.String(),
			},
			wantOutput: fmt.Sprintf("{\"keys\":[{\"crv\":\"P-256\",\"kid\":\"ver_[VERSION]\",\"kty\":\"EC\",\"x\":\"%s\",\"y\":\"%s\"},{\"crv\":\"P-256\",\"kid\":\"ver_[VERSION]2\",\"kty\":\"EC\",\"x\":\"%s\",\"y\":\"%s\"}]}",
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.X.Bytes()),
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.Y.Bytes()),
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.X.Bytes()),
				base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.Y.Bytes())),
		},
		{
			name:         "no-primary",
			storageState: map[string]string{},
			wantOutput:   "{\"keys\":[]}",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockKMSServer.KeyName = key
			mockKMSServer.Labels = make(map[string]string)
			for key, val := range tc.storageState {
				mockKMSServer.Labels[key] = val
			}

			keys, err := ks.JWKList(ctx, key)
			got, err := FormatJWKString(keys)
			testutil.ErrCmp(t, tc.wantErr, err)

			if err != nil {
				return
			}
			if diff := cmp.Diff(tc.wantOutput, got); diff != "" {
				t.Errorf("Got diff (-want, +got): %v", diff)
			}
		})
	}

}
