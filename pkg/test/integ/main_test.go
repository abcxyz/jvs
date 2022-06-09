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
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/google/uuid"
	"github.com/sethvargo/go-retry"
	"google.golang.org/api/iterator"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
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

	// TODO: Actual tests and stuff
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
