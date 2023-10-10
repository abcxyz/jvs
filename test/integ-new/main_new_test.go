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

// This folder (test/integ-new) will be used to replace
// folder test/integ after all new integ tests are implemented.
// For now these tests will be ran by ci-new.yml.
// If you are making changes to this file, please manully
// run the ci-new workflow to make sure things works.

// Main entry point for integration tests.
package integration

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/abcxyz/jvs/pkg/cli"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/api/iterator"
	grpccodes "google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// Global integration test config.
var (
	cfg *config

	httpClient *http.Client

	// Keys we don't compare in validation result.
	ignoreKeysMap map[string]struct{} = map[string]struct{}{
		"nbf": {},
		"jti": {},
	}

	// The keys in token map where their values is of type time.Time.
	tokenTimeKeysMap map[string]struct{} = map[string]struct{}{
		"exp": {},
		"iat": {},
	}
)

const (
	timestampFormat = "2006-01-02 3:04PM UTC"
	testTTLString   = "30m"
	testTTLTime     = 30 * time.Minute
)

func TestMain(m *testing.M) {
	os.Exit(func() int {
		ctx := context.Background()

		if strings.ToLower(os.Getenv("TEST_INTEGRATION")) != "true" {
			log.Printf("skipping (not integration)")
			// Not integration test. Exit.
			return 0
		}

		// set up global test config.
		c, err := newTestConfig(ctx)
		if err != nil {
			log.Printf("Failed to parse integration test config: %v", err)
			return 2
		}
		cfg = c

		httpClient = &http.Client{
			Timeout: 5 * time.Second,
		}

		return m.Run()
	}())
}

func TestAPIAndPublicKeyService(t *testing.T) {
	t.Parallel()

	ts := time.Now().UTC()

	cases := []struct {
		name          string
		isBreakglass  bool
		justification string
		wantTokenMap  map[string]any
	}{
		{
			name:          "none_breakglass",
			justification: "issues/12345",
			isBreakglass:  false,
			wantTokenMap: map[string]any{
				"aud": []string{"dev.abcxyz.jvs"},
				"exp": ts.Add(testTTLTime).UTC(),
				"iat": ts.UTC(),
				"iss": "jvs.abcxyz.dev",
				"jti": "",
				"justs": []any{
					map[string]any{
						"category": "explanation",
						"value":    "issues/12345",
					},
				},
				"nbf": "",
				"req": cfg.ServiceAccount,
				"sub": cfg.ServiceAccount,
			},
		},
		{
			name:          "breakglass",
			justification: "issues/12345",
			isBreakglass:  true,
			wantTokenMap: map[string]any{
				"aud": []string{"dev.abcxyz.jvs"},
				"exp": ts.Add(testTTLTime).UTC(),
				"iat": ts.UTC(),
				"iss": "jvsctl",
				"jti": "",
				"justs": []any{
					map[string]any{
						"category": "breakglass",
						"value":    "issues/12345",
					},
				},
				"nbf": "",
				"sub": "",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			var createCmd cli.TokenCreateCommand
			_, stdout, _ := createCmd.Pipe()

			createTokenArgs := []string{"-server", cfg.APIServer, "-justification", tc.justification, "-ttl", testTTLString, "--auth-token", cfg.APIServiceIDToken}
			if tc.isBreakglass {
				createTokenArgs = append(createTokenArgs, "-breakglass")
			}

			if err := createCmd.Run(ctx, createTokenArgs); err != nil {
				t.Fatalf("failed to create token: %v", err)
			}

			token := stdout.String()

			parsedToken, err := jwt.ParseInsecure([]byte(token))
			if err != nil {
				t.Fatalf("failed to parse token: %v", err)
			}

			parsedTokenMap, err := parsedToken.AsMap(ctx)
			if err != nil {
				t.Fatalf("failed to parse token to map: %v", err)
			}

			validateTokenArgs := []string{"-token", token, "-jwks-endpoint", cfg.JWKSEndpoint, "--format", "json"}

			var validateCmd cli.TokenValidateCommand
			_, _, _ = validateCmd.Pipe()

			if err := validateCmd.Run(ctx, validateTokenArgs); err != nil {
				t.Fatalf("jvs service failed to validate token: %v", err)
			}

			if diff := cmp.Diff(testNormalizeTokenMap(t, tc.wantTokenMap), testNormalizeTokenMap(t, parsedTokenMap)); diff != "" {
				t.Errorf("token got unexpected diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestUIServiceHealthCheck(t *testing.T) {
	t.Parallel()

	addr := cfg.UIServiceAddr
	wantStatusCode := http.StatusOK
	healthCheckPath := "/health"

	ctx := context.Background()

	uri := addr + healthCheckPath
	resp := testSendHTTPReq(ctx, t, uri, cfg.UIServiceIDToken)

	defer resp.Body.Close()

	if got, want := resp.StatusCode, wantStatusCode; got != want {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("health check status code got=%d want=%d, response=%s", got, want, string(b))
	}
}

func TestCertRotatorHealthCheck(t *testing.T) {
	t.Parallel()

	healthCheckPath := "/health"
	wantStatusCode := http.StatusOK

	ctx := context.Background()

	uri := cfg.CertRotatorServiceAddr + healthCheckPath
	resp := testSendHTTPReq(ctx, t, uri, cfg.CertRotatorServiceIDToken)

	defer resp.Body.Close()

	if got, want := resp.StatusCode, wantStatusCode; got != want {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("Got unexpected status code, got=%d want=%d, response=%s", got, want, string(b))
	}
}

// Subtests must be run in sequence, and they have waits in between.
// Therefore, they cannot be parallelized, and aren't a good fit for table testing.
//
//nolint:paralleltest
func TestCertRotatorKeyRotation(t *testing.T) {
	t.Parallel()
	testutil.SkipIfNotIntegration(t)

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		t.Fatalf("failed to setup kms client: %s", err)
	}

	keyResouceName := fmt.Sprintf("projects/%s/locations/global/keyRings/%s/cryptoKeys/%s", cfg.ProjectID, cfg.KeyRing, cfg.KeyName)

	// Validate we have a single enabled key that is primary.
	testValidateKeyVersionState(ctx, t, kmsClient, keyResouceName, 1,
		map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
			1: kmspb.CryptoKeyVersion_ENABLED,
		})

	t.Run("new_key_creation", func(t *testing.T) {
		time.Sleep(5001 * time.Millisecond) // Wait past the next rotation event

		testCallKeyRotationEndpoint(ctx, t)

		// Validate we have created a new key, but haven't set it as primary yet.
		testValidateKeyVersionState(ctx, t, kmsClient, keyResouceName, 1,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_ENABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	t.Run("new_key_promotion", func(t *testing.T) {
		time.Sleep(1001 * time.Millisecond) // Wait past the propagation delay.

		testCallKeyRotationEndpoint(ctx, t)

		// Validate our new key has been set to primary
		testValidateKeyVersionState(ctx, t, kmsClient, keyResouceName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_ENABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	t.Run("old_key_disable", func(t *testing.T) {
		time.Sleep(2001 * time.Millisecond) // Wait past the grace period.

		testCallKeyRotationEndpoint(ctx, t)

		// Validate that our old key has been disabled.
		testValidateKeyVersionState(ctx, t, kmsClient, keyResouceName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_DISABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	t.Run("old_key_destroy", func(t *testing.T) {
		time.Sleep(2001 * time.Millisecond) // Wait past the disabled period and next rotation event.

		testCallKeyRotationEndpoint(ctx, t)

		// Validate that our old key has been scheduled for destruction, and cycle has started again.
		testValidateKeyVersionState(ctx, t, kmsClient, keyResouceName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_DESTROY_SCHEDULED,
				2: kmspb.CryptoKeyVersion_ENABLED,
				3: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	t.Cleanup(func() {
		testCleanUpKey(ctx, t, kmsClient, keyResouceName)
		if err := kmsClient.Close(); err != nil {
			t.Errorf("Clean up of key %s failed: %s", keyResouceName, err)
		}
	})
}

// testNormalizeTokenMap parses the tokenMap by overwriting the value for ignoreKeys
// to empty and parsing the timestamp to the format without seconds to make
// test on timestamps less flaky.
func testNormalizeTokenMap(tb testing.TB, m map[string]any) map[string]any {
	tb.Helper()

	for k, v := range m {
		// set the values of ignored key to empty
		if _, ok := ignoreKeysMap[k]; ok {
			m[k] = ""
			continue
		}

		// format timestamp
		if _, ok := tokenTimeKeysMap[k]; ok {
			ts, ok := v.(time.Time)
			if !ok {
				tb.Fatalf("failed to parse %v to time.Time", v)
			}
			m[k] = ts.Format(timestampFormat)
		}
	}
	return m
}

func testSendHTTPReq(ctx context.Context, tb testing.TB, uri, token string) *http.Response {
	tb.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		tb.Fatalf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := httpClient.Do(req)
	if err != nil {
		tb.Fatalf("client failed to get response: %v", err)
	}

	return resp
}

func testCallKeyRotationEndpoint(ctx context.Context, tb testing.TB) {
	tb.Helper()

	path := "/"
	wantStatusCode := http.StatusOK

	uri := cfg.CertRotatorServiceAddr + path

	resp := testSendHTTPReq(ctx, tb, uri, cfg.CertRotatorServiceIDToken)

	defer resp.Body.Close()

	if got, want := resp.StatusCode, wantStatusCode; got != want {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			tb.Fatal(err)
		}
		tb.Errorf("Got unexpected status code, got=%d want=%d, response=%s", got, want, string(b))
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
	primaryName := resp.Labels[jvscrypto.PrimaryKey]
	primaryLabel := primaryName[strings.LastIndex(primaryName, "/")+1:]
	primaryNumber, err := strconv.Atoi(strings.TrimPrefix(primaryLabel, jvscrypto.PrimaryLabelPrefix))
	if err != nil {
		tb.Fatalf("couldn't convert version %s to number: %s", primaryName, err)
	}
	if primaryNumber != expectedPrimary {
		tb.Errorf("primary was set to version %d, but expected %d", primaryNumber, expectedPrimary)
	}
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
			// Cloud KMS returns the following errors when the key is already
			// destroyed or does not exist.
			code := grpcstatus.Code(err)
			if code == grpccodes.NotFound || code == grpccodes.FailedPrecondition {
				return
			}

			tb.Errorf("cleanup: failed to destroy crypto key version %q: %s", ver.Name, err)
		}
	}
}
