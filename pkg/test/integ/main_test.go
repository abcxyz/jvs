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
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/abcxyz/pkg/cache"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
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
			name: "no_ttl",
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

func TestPublicKeys(t *testing.T) {
	ctx := context.Background()
	if !testIsIntegration(t) {
		// Not an integ test, don't run anything.
		t.Skip("Not an integration test, skipping...")
		return
	}

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		t.Fatalf("failed to setup kms client: %s", err)
	}

	config, err := config.LoadPublicKeyConfig(ctx, []byte{})
	if err != nil {
		t.Fatalf("failed to load public key config: %s", err)
	}

	cache := cache.New[string](config.CacheTimeout)

	ks := &jvscrypto.KeyServer{
		KMSClient:       kmsClient,
		PublicKeyConfig: config,
		Cache:           cache,
	}

	//test for no-primary
	testValidatePublicKeys(ctx, t, kmsClient, ks, "")

	keyRing := os.Getenv("TEST_JVS_KMS_KEY_RING")
	if keyRing == "" {
		t.Fatal("Key ring must be provided using TEST_JVS_KMS_KEY_RING env variable.")
	}
	if err != nil {
		t.Fatalf("failed to setup kms client: %s", err)
	}

	keyRing = strings.Trim(keyRing, "\"")
	keyName := testCreateKey(ctx, t, kmsClient, keyRing)

	//test for one key version
	testValidatePublicKeys(ctx, t, kmsClient, ks, keyName)

	testCreateKeyVersion(ctx, t, kmsClient, keyName, "2")
	//test for multiple key version
	testValidatePublicKeys(ctx, t, kmsClient, ks, keyName)
	t.Cleanup(func() {
		testCleanUpKey(ctx, t, kmsClient, keyName)
		err := kmsClient.Close()
		if err != nil {
			t.Errorf("Clean up of key %s failed: %s", keyName, err)
		}
	})
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

// Create a new KeyVersion for use with integration tests.
func testCreateKeyVersion(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName, keyVersion string) string {
	tb.Helper()
	ck, err := kmsClient.CreateCryptoKeyVersion(ctx, &kmspb.CreateCryptoKeyVersionRequest{
		Parent:           keyName,
		CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
	})
	if err != nil {
		tb.Fatalf("failed to create crypto keyVersion: %s", err)
	}

	// Wait for a key version to be created and enabled.
	r := retry.NewExponential(100 * time.Millisecond)
	if err := retry.Do(ctx, retry.WithMaxRetries(10, r), func(ctx context.Context) error {
		ckv, err := kmsClient.GetCryptoKeyVersion(ctx, &kmspb.GetCryptoKeyVersionRequest{
			Name: ck.Name + "/cryptoKeyVersions/" + keyVersion,
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

// Build a key set containing the public key for the first key version from the specified key. Static, does not update automatically.
func testKeySetFromKMS(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName string) jwk.Set {
	tb.Helper()
	keySet := jwk.NewSet()
	versionName := keyName + "/cryptoKeyVersions/1"
	pubKeyResp, err := kmsClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: versionName})
	if err != nil {
		tb.Fatalf("Couldn't retrieve public key: %s", err)
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

// Creates a JWK Set(including public keys) converted to string.
func testPublicKeysFromKMS(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName string) (string, error) {
	tb.Helper()
	it := kmsClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: keyName,
		Filter: "state=ENABLED",
	})

	jwkList := make([]*jvscrypto.ECDSAKey, 0)
	for {
		// Could parallelize this. #34
		ver, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("err while reading crypto key version list: %w", err)
		}
		key, err := kmsClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: ver.Name})
		if err != nil {
			return "", fmt.Errorf("err while getting public key from kms: %w", err)
		}

		block, _ := pem.Decode([]byte(key.Pem))
		if block == nil || block.Type != "PUBLIC KEY" {
			return "", fmt.Errorf("failed to decode PEM block containing public key")
		}

		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return "", fmt.Errorf("failed to parse public key")
		}

		ecdsaKey, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return "", fmt.Errorf("unknown key format, expected ecdsa, got %T", pub)
		}
		if len(ecdsaKey.X.Bits()) == 0 || len(ecdsaKey.Y.Bits()) == 0 {
			return "", fmt.Errorf("unable to determine X and/or Y for ECDSA key")
		}
		ek := &jvscrypto.ECDSAKey{
			Curve: "P-256",
			ID:    ver.Name,
			Type:  "EC",
			X:     base64.RawURLEncoding.EncodeToString(ecdsaKey.X.Bytes()),
			Y:     base64.RawURLEncoding.EncodeToString(ecdsaKey.Y.Bytes()),
		}
		jwkList = append(jwkList, ek)
	}
	sort.Slice(jwkList, func(i, j int) bool {
		return (*jwkList[i]).ID < (*jwkList[j]).ID
	})
	jwks := &jvscrypto.JWKS{Keys: jwkList}
	json, err := json.Marshal(jwks)
	if err != nil {
		return "", fmt.Errorf("err while converting jwk to json: %w", err)
	}
	return string(json), nil
}

func testValidatePublicKeys(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, ks *jvscrypto.KeyServer, keyName string,
) {
	tb.Helper()
	expectedPublicKeys, err := testPublicKeysFromKMS(ctx, tb, kmsClient, keyName)
	if err != nil {
		tb.Fatalf("failed to get public keys from KMS")
	}
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		tb.Fatalf("http.NewRequest(): got %v, want no error", err)
	}

	rw := httptest.NewRecorder()
	ks.ServeHTTP(rw, req)
	if gotCode, wantCode := rw.Code, http.StatusOK; gotCode != wantCode {
		tb.Errorf("rw.Code: got %d, want %d", gotCode, wantCode)
		return
	}
	got := rw.Body.String()
	if diff := cmp.Diff(expectedPublicKeys, got); diff != "" {
		tb.Errorf("GotPublicKeys diff (-want, +got): %v", diff)
	}
}
