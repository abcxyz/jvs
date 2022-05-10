package jvscrypto

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
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

func TestGetJWKSetFormattedString(t *testing.T) {
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
		StateStore:   &KeyLabelStateStore{KmsClient: kms},
	}

	key := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s", "[PROJECT]", "[LOCATION]", "[KEY_RING]", "[CRYPTO_KEY]")
	versionName := fmt.Sprintf("%s/cryptoKeyVersions/[VERSION]", key)

	tests := []struct {
		name       string
		wantOutput string
		wantErr    string
	}{
		{
			name:       "happy-path",
			wantOutput: "err",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mockKMSServer.KeyName = key
			mockKMSServer.Labels = make(map[string]string)
			mockKMSServer.Labels[versionName] = VersionStatePrimary.String()
			got, err := ks.getJWKSetFormattedString(ctx, key)
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
