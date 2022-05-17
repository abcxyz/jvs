package justification

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/golang-jwt/jwt"
	"github.com/golang/protobuf/proto"
	"github.com/sethvargo/go-gcpkms/pkg/gcpkms"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestCreateToken(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var clientOpt option.ClientOption
	var mockKeyManagement = &testutil.MockKeyManagementServer{
		UnimplementedKeyManagementServiceServer: kmspb.UnimplementedKeyManagementServiceServer{},
		Reqs:                                    make([]proto.Message, 1),
		Err:                                     nil,
		Resps:                                   make([]proto.Message, 1),
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	mockKeyManagement.PrivateKey = privateKey
	x509EncodedPub, err := x509.MarshalPKIXPublicKey(privateKey.Public())
	if err != nil {
		t.Fatal(err)
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})
	mockKeyManagement.PublicKey = string(pemEncodedPub)

	serv := grpc.NewServer()
	kmspb.RegisterKeyManagementServiceServer(serv, mockKeyManagement)

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

	c, err := kms.NewKeyManagementClient(ctx, clientOpt)
	if err != nil {
		t.Fatal(err)
	}

	signer, err := gcpkms.NewSigner(ctx, c, "keyName")
	if err != nil {
		t.Fatal(err)
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
			mockKeyManagement.Reqs = nil
			mockKeyManagement.Err = tc.serverErr

			mockKeyManagement.Resps = append(mockKeyManagement.Resps[:0], &kmspb.CryptoKeyVersion{})

			response, gotErr := processor.CreateToken(ctx, tc.request)
			testutil.ErrCmp(t, tc.wantErr, gotErr)

			if gotErr != nil {
				return
			}
			if err := jvscrypto.VerifyJWTString(ctx, c, "keyName", response); err != nil {
				t.Errorf("Unable to verify signed jwt. %v", err)
			}

			claims := &jvspb.JVSClaims{}
			if _, err := jwt.ParseWithClaims(response, claims, func(token *jwt.Token) (interface{}, error) {
				return privateKey.Public(), nil
			}); err != nil {
				t.Errorf("Unable to parse created jwt string. %v", err)
			}
			validateClaims(t, claims, tc.request.Justifications)
		})
	}
}

func validateClaims(t testing.TB, provided *jvspb.JVSClaims, expectedJustifications []*jvspb.Justification) {
	// test the standard claims filled by processor
	if provided.StandardClaims.Issuer != jvsIssuer {
		t.Errorf("audience value %s incorrect, expected %s", provided.StandardClaims.Issuer, jvsIssuer)
	}
	// TODO: as we add more standard claims, add more validations.

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
			return
		}
	}
}
