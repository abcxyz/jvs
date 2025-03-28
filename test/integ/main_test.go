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
package integration

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"github.com/abcxyz/jvs/pkg/cli"
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
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

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

	healthCheckPath := "/health"

	ctx := t.Context()

	uri := cfg.UIServiceAddr + healthCheckPath

	testCallEndpoint(ctx, t, uri, cfg.UIServiceIDToken, http.StatusOK)
}

func TestCertRotatorHealthCheck(t *testing.T) {
	t.Parallel()

	healthCheckPath := "/health"

	ctx := t.Context()

	uri := cfg.CertRotatorServiceAddr + healthCheckPath

	testCallEndpoint(ctx, t, uri, cfg.CertRotatorServiceIDToken, http.StatusOK)
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

func testCallEndpoint(ctx context.Context, tb testing.TB, uri, token string, wantStatusCode int) {
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

	defer resp.Body.Close()

	if got, want := resp.StatusCode, wantStatusCode; got != want {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			tb.Fatal(err)
		}
		tb.Errorf("Got unexpected status code, got=%d want=%d, response=%s", got, want, string(b))
	}
}
