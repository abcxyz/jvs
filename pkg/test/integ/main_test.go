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
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestJVS(t *testing.T) {
	t.Parallel()
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

func TestRotator(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	if !testIsIntegration(t) {
		// Not an integ test, don't run anything.
		t.Skip("Not an integration test, skipping...")
		return
	}

	kmsClient, keyName := testSetupRotator(ctx, t)

	cfg := &config.CryptoConfig{
		Version:          1,
		KeyTTL:           7 * time.Second,
		GracePeriod:      2 * time.Second, // rotate after 5 seconds
		PropagationDelay: time.Second,
		DisabledPeriod:   time.Second,
		KeyNames:         []string{keyName},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	r := &jvscrypto.RotationHandler{
		KMSClient:    kmsClient,
		CryptoConfig: cfg,
	}

	// Validate we have a single enabled key that is primary.
	testValidateKeyVersionState(ctx, t, kmsClient, keyName, 1,
		map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
			1: kmspb.CryptoKeyVersion_ENABLED,
		})

	// These tests must be run in sequence, and they have waits in between. Therefore, they cannot
	// be parallelized, and aren't a good fit for table testing.

	t.Run("new_key_creation", func(t *testing.T) {
		time.Sleep(5001 * time.Millisecond) // Wait past the next rotation event
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
			return
		}
		time.Sleep(50 * time.Millisecond) // Reduces chance key will be in "pending generation" state
		// Validate we have created a new key, but haven't set it as primary yet.
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 1,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_ENABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	t.Run("new_key_promotion", func(t *testing.T) {
		time.Sleep(1001 * time.Millisecond) // Wait past the propagation delay.
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}
		// Validate our new key has been set to primary
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_ENABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	t.Run("old_key_disable", func(t *testing.T) {
		time.Sleep(2001 * time.Millisecond) // Wait past the grace period.
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}
		// Validate that our old key has been disabled.
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_DISABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	t.Run("old_key_destroy", func(t *testing.T) {
		time.Sleep(2001 * time.Millisecond) // Wait past the disabled period and next rotation event.
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}
		time.Sleep(50 * time.Millisecond) // Reduces chance key will be in "pending generation" state
		// Validate that our old key has been scheduled for destruction, and cycle has started again.
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_DESTROY_SCHEDULED,
				2: kmspb.CryptoKeyVersion_ENABLED,
				3: kmspb.CryptoKeyVersion_ENABLED,
			})
	})
}

func TestRotator_EdgeCases(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	if !testIsIntegration(t) {
		// Not an integ test, don't run anything.
		t.Skip("Not an integration test, skipping...")
		return
	}

	kmsClient, keyName := testSetupRotator(ctx, t)

	cfg := &config.CryptoConfig{
		Version:          1,
		KeyTTL:           99 * time.Hour,
		GracePeriod:      time.Second,
		PropagationDelay: time.Second,
		DisabledPeriod:   time.Second,
		KeyNames:         []string{keyName},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	r := &jvscrypto.RotationHandler{
		KMSClient:    kmsClient,
		CryptoConfig: cfg,
	}

	time.Sleep(1001 * time.Millisecond) // Wait past the propagation delay.

	t.Run("invalid_primary", func(t *testing.T) {
		// Set primary to a version that doesn't exist
		if err := jvscrypto.SetPrimary(ctx, kmsClient, keyName, keyName+"/cryptoKeyVersions/99"); err != nil {
			t.Fatalf("unable to set primary: %s", err)
		}
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}

		// Validate that we fixed the situation by setting our valid key to primary
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 1,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	// we could parallelize this, but we'd need separate keys from the above (more cruft)
	t.Run("emergent_disable", func(t *testing.T) {
		// Emergently disable our primary.
		testEmergentDisable(ctx, t, kmsClient, keyName, keyName+"/cryptoKeyVersions/1")

		// Validate that the rotator will fix the situation by creating a new version and setting it to primary
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}
		time.Sleep(50 * time.Millisecond) // Reduces chance key will be in "pending generation" state
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_DISABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	})
}

// Set up KMS, create a key, and set the primary.
func testSetupRotator(ctx context.Context, tb testing.TB) (*kms.KeyManagementClient, string) {
	tb.Helper()
	keyRing := os.Getenv("TEST_JVS_KMS_KEY_RING")
	if keyRing == "" {
		tb.Fatal("Key ring must be provided using TEST_JVS_KMS_KEY_RING env variable.")
	}

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		tb.Fatalf("failed to setup kms client: %s", err)
	}

	keyRing = strings.Trim(keyRing, "\"")
	keyName := testCreateKey(ctx, tb, kmsClient, keyRing)
	tb.Cleanup(func() {
		testCleanUpKey(ctx, tb, kmsClient, keyName)
		err := kmsClient.Close()
		if err != nil {
			tb.Errorf("Clean up of key %s failed: %s", keyName, err)
		}
	})
	if err := jvscrypto.SetPrimary(ctx, kmsClient, keyName, keyName+"/cryptoKeyVersions/1"); err != nil {
		tb.Fatalf("unable to set primary: %s", err)
	}

	// Validate we have a single enabled key that is primary.
	testValidateKeyVersionState(ctx, tb, kmsClient, keyName, 1,
		map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
			1: kmspb.CryptoKeyVersion_ENABLED,
		})

	return kmsClient, keyName
}

// This is intended to mock an event where we need to emergently rotate the key.
// We disable the key version and remove it as primary.
func testEmergentDisable(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName string, versionName string) {
	tb.Helper()
	ver, err := kmsClient.GetCryptoKeyVersion(ctx, &kmspb.GetCryptoKeyVersionRequest{Name: versionName})
	if err != nil {
		tb.Fatalf("unable to retreive version %s: %s", versionName, err)
	}

	ver.State = kmspb.CryptoKeyVersion_DISABLED
	var messageType *kmspb.CryptoKeyVersion
	mask, err := fieldmaskpb.New(messageType, "state")
	if err != nil {
		tb.Fatalf("unable to create field mask: %s", err)
	}
	updateReq := &kmspb.UpdateCryptoKeyVersionRequest{
		CryptoKeyVersion: ver,
		UpdateMask:       mask,
	}
	if _, err := kmsClient.UpdateCryptoKeyVersion(ctx, updateReq); err != nil {
		tb.Fatalf("unable to disable version: %s", err)
	}

	key, err := kmsClient.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{Name: keyName})
	if err != nil {
		tb.Fatalf("unable to retreive key %s: %s", keyName, err)
	}

	labels := make(map[string]string, 0)
	key.Labels = labels

	var mT *kmspb.CryptoKey
	mask, err = fieldmaskpb.New(mT, "labels")
	if err != nil {
		tb.Fatalf("unable to create field mask: %s", err)
	}
	if _, err = kmsClient.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{CryptoKey: key, UpdateMask: mask}); err != nil {
		tb.Fatalf("unable to set labels: %s", err)
	}
}

func testValidateKeyVersionState(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName string,
	expectedPrimary int, expectedStates map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState,
) {
	tb.Helper()
	// validate that each version is in the expected state.
	it := kmsClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: keyName,
	})
	count := 0
	for {
		ver, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			tb.Fatalf("err while calling kms to get key versions: %s", err)
		}
		number, err := strconv.Atoi(ver.Name[strings.LastIndex(ver.Name, "/")+1:])
		if err != nil {
			tb.Fatalf("couldn't convert version %s to number: %s", ver.Name, err)
		}
		if ver.State != expectedStates[number] {
			tb.Errorf("version %s was in state %s, but expected %s", ver.Name, ver.State, expectedStates[number])
		}
		count++
	}
	if count != len(expectedStates) {
		tb.Errorf("got %d versions, expected %d", count, len(expectedStates))
	}

	// validate the primary is set correctly.
	resp, err := kmsClient.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{Name: keyName})
	if err != nil {
		tb.Fatalf("err while calling kms: %s", err)
	}
	primaryName := resp.Labels["primary"]
	primaryLabel := primaryName[strings.LastIndex(primaryName, "/")+1:]
	primaryNumber, err := strconv.Atoi(strings.TrimPrefix(primaryLabel, "ver_"))
	if err != nil {
		tb.Fatalf("couldn't convert version %s to number: %s", primaryName, err)
	}
	if primaryNumber != expectedPrimary {
		tb.Errorf("primary was set to version %d, but expected %d", primaryNumber, expectedPrimary)
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
