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

// Main entry point for integration tests.
package integ

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/sethvargo/go-retry"
	"google.golang.org/api/iterator"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestJVS(t *testing.T) {
	ctx := context.Background()
	if !testIsIntegration(t) {
		// Not an integ test, don't run anything.
		t.Skip("Not an integration test, skipping...")
		return
	}
	keyRing := os.Getenv("TEST_JVS_KMS_KEY_RING")
	if keyRing == "" {
		t.Fatal("Key ring must be provided using TEST_JVS_KMS_KEY_RING env variable.")
	}

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		t.Fatalf("failed to setup kms client: %s", err)
	}

	keyRing = strings.Trim(keyRing, "\"")
	keyName := testCreateKey(ctx, t, kmsClient, keyRing)
	t.Cleanup(func() {
		testCleanUpKey(ctx, t, kmsClient, keyName)
		err := kmsClient.Close()
		if err != nil {
			t.Errorf("Clean up of key %s failed: %s", keyName, err)
		}
	})

	cfg := &config.JustificationConfig{
		Version: 1,
		KeyName: keyName,
		Issuer:  "ci-test",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	p := justification.NewProcessor(kmsClient, cfg)
	jvsAgent := justification.NewJVSAgent(p)

	tests := []struct {
		name     string
		request  *jvspb.CreateJustificationRequest
		wantErr  string
		wantResp map[string]interface{}
	}{
		{
			name: "happy_path",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "This is a test.",
					},
				},
				Ttl: &durationpb.Duration{
					Seconds: 3600,
				},
			},
			wantResp: map[string]interface{}{
				"aud": []string{"TODO #22"},
				"iss": "ci-test",
				"justs": []any{
					map[string]any{"category": "explanation", "value": "This is a test."},
				},
				"sub": "TODO #22",
			},
		},
		{
			name: "unknown_justification",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "who-knows",
						Value:    "This is a test.",
					},
				},
				Ttl: &durationpb.Duration{
					Seconds: 3600,
				},
			},
			wantErr: "couldn't validate request",
		},
		{
			name: "blank_justification",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "",
					},
				},
				Ttl: &durationpb.Duration{
					Seconds: 3600,
				},
			},
			wantErr: "couldn't validate request",
		},
		{
			name: "no-_tl",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "This is a test.",
					},
				},
			},
			wantErr: "couldn't validate request",
		},
		{
			name: "no_justification",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{},
				Ttl: &durationpb.Duration{
					Seconds: 3600,
				},
			},
			wantErr: "couldn't validate request",
		},
		{
			name:    "empty_request",
			request: &jvspb.CreateJustificationRequest{},
			wantErr: "couldn't validate request",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resp, gotErr := jvsAgent.CreateJustification(ctx, tc.request)
			testutil.ErrCmp(t, tc.wantErr, gotErr)
			if gotErr != nil {
				return
			}

			keySet := testKeySetFromKMS(ctx, t, kmsClient, keyName)
			token, err := jvscrypto.ValidateJWT(keySet, resp.Token)
			if err != nil {
				t.Errorf("Couldn't validate signed token: %s", err)
				return
			}

			tokenMap, err := (*token).AsMap(ctx)
			if err != nil {
				t.Errorf("Couldn't convert token to map: %s", err)
				return
			}

			// These fields are set based on time, and we cannot know what they will be set to.
			ignoreFields := map[string]interface{}{
				"exp": nil,
				"iat": nil,
				"jti": nil,
				"nbf": nil,
			}
			ignoreOpt := cmpopts.IgnoreMapEntries(func(k string, v interface{}) bool { _, ok := ignoreFields[k]; return ok })
			if diff := cmp.Diff(tc.wantResp, tokenMap, ignoreOpt); diff != "" {
				t.Errorf("Got diff (-want, +got): %v", diff)
			}
		})
	}
}

func testIsIntegration(tb testing.TB) bool {
	tb.Helper()
	integVal := os.Getenv("TEST_JVS_INTEGRATION")
	if integVal == "" {
		return false
	}
	isInteg, err := strconv.ParseBool(integVal)
	if err != nil {
		tb.Fatalf("Unable to parse TEST_JVS_INTEGRATION flag %s: %s", integVal, err)
	}
	return isInteg
}

// Create an asymmetric signing key for use with integration tests.
func testCreateKey(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyRing string) string {
	tb.Helper()
	u, err := uuid.NewUUID()
	if err != nil {
		tb.Fatalf("failed to create uuid : %s", err)
	}
	ck, err := kmsClient.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{
		Parent:      keyRing,
		CryptoKeyId: u.String(),
		CryptoKey: &kmspb.CryptoKey{
			Purpose: kmspb.CryptoKey_ASYMMETRIC_SIGN,
			VersionTemplate: &kmspb.CryptoKeyVersionTemplate{
				Algorithm: kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256,
			},
		},
	})
	if err != nil {
		tb.Fatalf("failed to create crypto key: %s", err)
	}

	// Wait for a key version to be created and enabled.
	r := retry.NewExponential(100 * time.Millisecond)
	if err := retry.Do(ctx, retry.WithMaxRetries(10, r), func(ctx context.Context) error {
		ckv, err := kmsClient.GetCryptoKeyVersion(ctx, &kmspb.GetCryptoKeyVersionRequest{
			Name: ck.Name + "/cryptoKeyVersions/1",
		})
		if err != nil {
			return err
		}
		if ckv.State == kmspb.CryptoKeyVersion_ENABLED {
			return nil
		}
		return errors.New("key is not in ready state")
	}); err != nil {
		tb.Fatal("key did not enter ready state")
	}
	return ck.Name
}

// Destroy all versions within a key in order to clean up tests.
func testCleanUpKey(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName string) {
	tb.Helper()
	it := kmsClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: keyName,
	})

	for {
		ver, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			tb.Fatalf("err while reading crypto key version list: %s", err)
		}
		if ver.State == kmspb.CryptoKeyVersion_DESTROYED ||
			ver.State == kmspb.CryptoKeyVersion_DESTROY_SCHEDULED {
			// no need to destroy again
			continue
		}
		if _, err := kmsClient.DestroyCryptoKeyVersion(ctx, &kmspb.DestroyCryptoKeyVersionRequest{
			Name: ver.Name,
		}); err != nil {
			tb.Errorf("cleanup: failed to destroy crypto key version %q: %s", ver.Name, err)
		}
	}
}

// Build a key set containing the public key for the first key version from the specified key. Static, does not update automatically.
func testKeySetFromKMS(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName string) jwk.Set {
	tb.Helper()
	keySet := jwk.NewSet()
	versionName := keyName + "/cryptoKeyVersions/1"
	pubKeyResp, err := kmsClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: versionName})
	if err != nil {
		tb.Fatalf("Couldn't retrieve public key %s", err)
	}
	block, _ := pem.Decode([]byte(pubKeyResp.Pem))
	if block == nil || block.Type != "PUBLIC KEY" {
		tb.Fatal("failed to decode PEM block containing public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		tb.Fatalf("Couldn't parse public key: %s", err)
	}

	jwkKey, err := jwk.FromRaw(pub)
	if err != nil {
		tb.Fatalf("Couldn't convert key to jwk: %s", err)
	}
	if err := jwkKey.Set("kid", versionName); err != nil {
		tb.Fatalf("Couldn't set key id: %s", err)
	}
	if err := keySet.AddKey(jwkKey); err != nil {
		tb.Fatalf("Couldn't add jwk to set: %s", err)
	}
	return keySet
}
