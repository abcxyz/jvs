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
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/abcxyz/jvs/pkg/cli"
	"github.com/google/go-cmp/cmp"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Global integration test config.
var (
	cfg *config

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

			createTokenArgs := []string{"-server", cfg.APIServer, "-justification", tc.justification, "-ttl", testTTLString, "--auth-token", cfg.IDToken}
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

			if diff := cmp.Diff(testParseTokenMap(t, tc.wantTokenMap), testParseTokenMap(t, parsedTokenMap)); diff != "" {
				t.Errorf("token got unexpacted diff (-want, +got):\n%s", diff)
			}
		})
	}
}

// testParseToken parses the tokenMap by overwriting the value for ignoreKeys
// to empty and parsing the timestamp to the format without seconds to make
// test on timestamps less flaky.
func testParseTokenMap(tb testing.TB, m map[string]any) map[string]any {
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
