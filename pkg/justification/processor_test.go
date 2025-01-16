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

package justification

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/abcxyz/pkg/logging"
	pkgtestutil "github.com/abcxyz/pkg/testutil"
)

type mockValidator struct {
	resp   *jvspb.ValidateJustificationResponse
	uiData *jvspb.UIData
	err    error
}

func (m *mockValidator) Validate(ctx context.Context, req *jvspb.ValidateJustificationRequest) (*jvspb.ValidateJustificationResponse, error) {
	return m.resp, m.err
}

func (m *mockValidator) GetUIData(ctx context.Context, req *jvspb.GetUIDataRequest) (*jvspb.UIData, error) {
	return m.uiData, m.err
}

func TestCreateToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		request            *jvspb.CreateJustificationRequest
		requestor          string
		validators         map[string]jvspb.Validator
		wantTTL            time.Duration
		wantSubject        string
		wantAudiences      []string
		wantErr            string
		wantJustifications []*jvspb.Justification
		serverErr          error
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
				Ttl: durationpb.New(3600 * time.Second),
			},
			wantTTL:       1 * time.Hour,
			wantAudiences: []string{DefaultAudience},
			wantJustifications: []*jvspb.Justification{
				{
					Category: "explanation",
					Value:    "test",
				},
			},
		},
		{
			name: "custom_subject",
			request: &jvspb.CreateJustificationRequest{
				Subject: "user@example.com",
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "test",
					},
				},
				Ttl: durationpb.New(3600 * time.Second),
			},
			wantSubject:   "user@example.com",
			wantTTL:       1 * time.Hour,
			wantAudiences: []string{DefaultAudience},
			wantJustifications: []*jvspb.Justification{
				{
					Category: "explanation",
					Value:    "test",
				},
			},
		},
		{
			name: "subject_inherits_requestor",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "test",
					},
				},
				Ttl: durationpb.New(3600 * time.Second),
			},
			requestor:     "requestor@example.com",
			wantSubject:   "requestor@example.com",
			wantTTL:       1 * time.Hour,
			wantAudiences: []string{DefaultAudience},
			wantJustifications: []*jvspb.Justification{
				{
					Category: "explanation",
					Value:    "test",
				},
			},
		},
		{
			name: "custom_audience",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "test",
					},
				},
				Ttl:       durationpb.New(3600 * time.Second),
				Audiences: []string{"aud1", "aud2"},
			},
			wantTTL:       1 * time.Hour,
			wantAudiences: []string{"aud1", "aud2"},
			wantJustifications: []*jvspb.Justification{
				{
					Category: "explanation",
					Value:    "test",
				},
			},
		},
		{
			name: "no_justification",
			request: &jvspb.CreateJustificationRequest{
				Ttl: durationpb.New(3600 * time.Second),
			},
			wantErr: "failed to validate request",
		},
		{
			name: "justification_explanation_empty",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
					},
				},
				Ttl: durationpb.New(3600 * time.Second),
			},
			wantErr: "failed to validate request: failed validation criteria with error [explanation cannot be empty] and warning []",
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
			wantTTL:       15 * time.Minute, // comes from default
			wantAudiences: []string{"dev.abcxyz.jvs"},
			wantJustifications: []*jvspb.Justification{
				{
					Category: "explanation",
					Value:    "test",
				},
			},
		},
		{
			name: "ttl_exceeds_max",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "test",
					},
				},
				Ttl: durationpb.New(10 * time.Hour),
			},
			wantErr: "requested ttl (10h) cannot be greater than max tll (1h)",
		},
		{
			name: "justifications_too_long",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    strings.Repeat("test", 4_000),
					},
				},
				Ttl: durationpb.New(10 * time.Hour),
			},
			wantErr: "must be less than 4000 bytes",
		},
		{
			name: "audiences_too_long",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "test",
					},
				},
				Audiences: []string{strings.Repeat("test", 1_000)},
				Ttl:       durationpb.New(10 * time.Hour),
			},
			wantErr: "must be less than 1000 bytes",
		},
		{
			name: "happy_path_with_validator",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "jira",
						Value:    "test",
					},
				},
				Ttl: durationpb.New(3600 * time.Second),
			},
			validators: map[string]jvspb.Validator{
				"jira": &mockValidator{
					resp: &jvspb.ValidateJustificationResponse{
						Valid: true,
						Annotation: map[string]string{
							"jira_issue_id":  "1234",
							"jira_issue_url": "https://example.atlassian.net/browse/ABCD",
						},
					},
					uiData: &jvspb.UIData{
						DisplayName: "Jira issue key",
						Hint:        "Jira issue key under JVS project",
					},
				},
			},
			wantTTL:       1 * time.Hour,
			wantAudiences: []string{DefaultAudience},
			wantJustifications: []*jvspb.Justification{
				{
					Category: "jira",
					Value:    "test",
					Annotation: map[string]string{
						"jira_issue_id":  "1234",
						"jira_issue_url": "https://example.atlassian.net/browse/ABCD",
					},
				},
			},
		},
		{
			name: "happy_path_with_unused_validator",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "test",
					},
				},
				Ttl: durationpb.New(3600 * time.Second),
			},
			validators: map[string]jvspb.Validator{
				"jira": &mockValidator{
					resp: &jvspb.ValidateJustificationResponse{
						Valid: false,
						Error: []string{"bad jira ticket"},
					},
				},
			},
			wantTTL:       1 * time.Hour,
			wantAudiences: []string{DefaultAudience},
			wantJustifications: []*jvspb.Justification{
				{
					Category: "explanation",
					Value:    "test",
				},
			},
		},
		{
			name: "failed_validator_criteria",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "jira",
						Value:    "test",
					},
				},
				Ttl: durationpb.New(3600 * time.Second),
			},
			validators: map[string]jvspb.Validator{
				"jira": &mockValidator{
					resp: &jvspb.ValidateJustificationResponse{
						Valid: false,
						Error: []string{"bad explanation"},
					},
				},
			},
			wantErr: status.Error(codes.InvalidArgument, "failed to validate request: failed validation criteria with error [bad explanation] and warning []").Error(),
		},
		{
			name: "validator_err",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "jira",
						Value:    "test",
					},
				},
				Ttl: durationpb.New(3600 * time.Second),
			},
			validators: map[string]jvspb.Validator{
				"jira": &mockValidator{
					err: fmt.Errorf("Cannot connect to validator"),
				},
			},
			wantErr: status.Error(codes.Internal, "unable to validate request").Error(),
		},
		{
			name: "validator_missing_ui_data",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "jira",
						Value:    "test",
					},
				},
				Ttl: durationpb.New(3600 * time.Second),
			},
			validators: map[string]jvspb.Validator{
				"jira": &mockValidator{
					resp: &jvspb.ValidateJustificationResponse{
						Valid: true,
					},
				},
			},
			wantTTL:       1 * time.Hour,
			wantAudiences: []string{DefaultAudience},
			wantJustifications: []*jvspb.Justification{
				{
					Category: "jira",
					Value:    "test",
				},
			},
		},
		{
			name: "missing_validator",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "jira",
						Value:    "test",
					},
				},
				Ttl: durationpb.New(3600 * time.Second),
			},
			wantErr: status.Error(codes.InvalidArgument, "failed to validate request: category \"jira\" is not supported").Error(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))
			now := time.Now().UTC()

			var clientOpt option.ClientOption
			key := "projects/[PROJECT]/locations/[LOCATION]/keyRings/[KEY_RING]/cryptoKeys/[CRYPTO_KEY]"
			version := key + "/cryptoKeyVersions/[VERSION]"
			keyID := key + "/cryptoKeyVersions/[VERSION]-0"

			mockKeyManagement := testutil.NewMockKeyManagementServer(key, version, jvscrypto.PrimaryLabelPrefix+"[VERSION]"+"-0")
			mockKeyManagement.NumVersions = 1

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

			_, conn := pkgtestutil.FakeGRPCServer(t, func(s *grpc.Server) {
				kmspb.RegisterKeyManagementServiceServer(s, mockKeyManagement)
			})

			clientOpt = option.WithGRPCConn(conn)
			t.Cleanup(func() {
				conn.Close()
			})

			c, err := kms.NewKeyManagementClient(ctx, clientOpt)
			if err != nil {
				t.Fatal(err)
			}

			authKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			ecdsaKey, err := jwk.FromRaw(authKey.PublicKey)
			if err != nil {
				t.Fatal(err)
			}

			if err := ecdsaKey.Set(jwk.KeyIDKey, keyID); err != nil {
				t.Fatal(err)
			}

			processor := NewProcessor(c, &config.JustificationConfig{
				KeyName:            key,
				SignerCacheTimeout: 5 * time.Minute,
				Issuer:             "test-iss",
				DefaultTTL:         1 * time.Minute,
				MaxTTL:             1 * time.Hour,
			}).WithValidators(tc.validators)

			mockKeyManagement.Reqs = nil
			mockKeyManagement.Err = tc.serverErr

			mockKeyManagement.Resps = append(mockKeyManagement.Resps[:0], &kmspb.CryptoKeyVersion{})

			response, gotErr := processor.CreateToken(ctx, tc.requestor, tc.request)
			if diff := pkgtestutil.DiffErrString(gotErr, tc.wantErr); diff != "" {
				t.Error(diff)
			}
			if gotErr != nil {
				return
			}

			// Validate message headers - we have to parse the full envelope for this.
			message, err := jws.Parse(response)
			if err != nil {
				t.Fatal(err)
			}
			sigs := message.Signatures()
			if got, want := len(sigs), 1; got != want {
				t.Errorf("expected length %d to be %d: %#v", got, want, sigs)
			} else {
				headers := sigs[0].ProtectedHeaders()
				if got, want := headers.Type(), "JWT"; got != want {
					t.Errorf("typ: expected %q to be %q", got, want)
				}
				if got, want := string(headers.Algorithm()), "ES256"; got != want {
					t.Errorf("alg: expected %q to be %q", got, want)
				}
				if got, want := headers.KeyID(), keyID; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			}

			// Parse as a JWT.
			token, err := jwt.Parse(response,
				jwt.WithContext(ctx),
				jwt.WithKey(jwa.ES256, privateKey.Public()),
				jwt.WithAcceptableSkew(5*time.Second),
				jvspb.WithTypedJustifications(),
			)
			if err != nil {
				t.Fatal(err)
			}

			// Validate standard claims.
			if got, want := token.Audience(), tc.wantAudiences; !reflect.DeepEqual(got, want) {
				t.Errorf("aud: expected %q to be %q", got, want)
			}
			if got := token.Expiration(); !got.After(now) {
				t.Errorf("exp: expected %q to be after %q (%q)", got, now, got.Sub(now))
			}
			if got := token.IssuedAt(); got.IsZero() {
				t.Errorf("iat: expected %q to be", got)
			}
			if got, want := token.Issuer(), "test-iss"; got != want {
				t.Errorf("iss: expected %q to be %q", got, want)
			}
			if got, want := len(token.JwtID()), 36; got != want {
				t.Errorf("jti: expected length %d to be %d: %#v", got, want, token.JwtID())
			}
			if got := token.NotBefore(); !got.Before(now) {
				t.Errorf("nbf: expected %q to be after %q (%q)", got, now, got.Sub(now))
			}
			if got, want := token.Subject(), tc.wantSubject; got != want {
				t.Errorf("sub: expected %q to be %q", got, want)
			}

			// Validate custom claims.
			gotRequestor, err := jvspb.GetRequestor(token)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := gotRequestor, tc.requestor; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}

			gotJustifications, err := jvspb.GetJustifications(token)
			if err != nil {
				t.Fatal(err)
			}
			expectedJustifications := tc.wantJustifications
			if diff := cmp.Diff(expectedJustifications, gotJustifications, cmpopts.IgnoreUnexported(jvspb.Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestComputeTTL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		req  time.Duration
		def  time.Duration
		max  time.Duration
		exp  time.Duration
		err  string
	}{
		{
			name: "request_zero_uses_default",
			req:  0,
			def:  15 * time.Minute,
			max:  30 * time.Minute,
			exp:  15 * time.Minute,
		},
		{
			name: "request_negative_uses_default",
			req:  -10 * time.Second,
			def:  15 * time.Minute,
			max:  30 * time.Minute,
			exp:  15 * time.Minute,
		},
		{
			name: "request_uses_self_in_bounds",
			req:  12 * time.Minute,
			def:  15 * time.Minute,
			max:  30 * time.Minute,
			exp:  12 * time.Minute,
		},
		{
			name: "request_greater_than_max_errors",
			req:  1 * time.Hour,
			def:  15 * time.Minute,
			max:  30 * time.Minute,
			err:  "requested ttl (1h) cannot be greater than max tll (30m)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := computeTTL(tc.req, tc.def, tc.max)
			if result := pkgtestutil.DiffErrString(err, tc.err); result != "" {
				t.Fatal(result)
			}

			if want := tc.exp; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}
