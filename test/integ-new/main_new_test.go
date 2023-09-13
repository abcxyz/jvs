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
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/abcxyz/jvs/pkg/cli"
	"github.com/google/go-cmp/cmp"
)

var (
	// Global integration test config.
	cfg *config

	// Keys we don't compare in validation result.
	ignoreKeysMap map[string]struct{}
)

const (
	serviceURLPrefix   = "https://"
	jwkEndpointPostfix = ".well-known/jwks"
	timestampFormat    = "2006-01-02 3:04PM UTC"
)

func TestMain(m *testing.M) {
	os.Exit(func() int {
		ctx := context.Background()

		ignoreKeysMap = map[string]struct{}{
			"nbf": {},
			"jti": {},
		}

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
	now := time.Now()
	cases := []struct {
		name                 string
		ttl                  string
		isBreakglass         bool
		justification        string
		wantValidationResult string
	}{
		{
			name:          "none_breakglass",
			justification: "issues/12345",
			ttl:           "30m",
			isBreakglass:  false,
			wantValidationResult: fmt.Sprintf(`
----- Justifications -----
explanation    issues/12345

----- Claims -----
aud    [dev.abcxyz.jvs]
exp    %s
iat    %s
iss    jvs.abcxyz.dev
req    %s
sub    %s
`, now.Add(30*time.Minute).UTC().Format(timestampFormat), now.Format(timestampFormat), cfg.ServiceAccount, cfg.ServiceAccount),
		},
		{
			name:          "breakglass",
			justification: "issues/12345",
			ttl:           "30m",
			isBreakglass:  true,
			wantValidationResult: fmt.Sprintf(`
Warning! This is a breakglass token.

----- Justifications -----
breakglass    issues/12345

----- Claims -----
aud    [dev.abcxyz.jvs]
exp    %s
iat    %s
iss    jvsctl
sub
`, now.Add(30*time.Minute).UTC().Format(timestampFormat), now.Format(timestampFormat)),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			var createCmd cli.TokenCreateCommand
			_, stdout, _ := createCmd.Pipe()

			server := fmt.Sprintf("%s---%s-%s:443", cfg.RevisionTagID, cfg.APIServiceName, cfg.ServiceURLPostfix)

			createTokenArgs := []string{"-server", server, "-justification", tc.justification, "--auth-token", cfg.IDToken}
			if tc.isBreakglass {
				createTokenArgs = append(createTokenArgs, "-breakglass")
			}

			if tc.ttl != "" {
				createTokenArgs = append(createTokenArgs, "-ttl", tc.ttl)
			}

			if err := createCmd.Run(ctx, createTokenArgs); err != nil {
				t.Fatal(err)
			}

			token := stdout.String()

			endpoint := fmt.Sprintf("%s%s---%s-%s/%s", serviceURLPrefix, cfg.RevisionTagID, cfg.PublicKeyServiceName, cfg.ServiceURLPostfix, jwkEndpointPostfix)
			validateTokenArgs := []string{"-token", token, "-jwks-endpoint", endpoint}

			var validateCmd cli.TokenValidateCommand
			_, stdout, _ = validateCmd.Pipe()

			if err := validateCmd.Run(ctx, validateTokenArgs); err != nil {
				t.Fatal(err)
			}

			gotValidationResult := stdout.String()
			if diff := cmp.Diff(testParseValidateResult(t, gotValidationResult, ignoreKeysMap), testParseValidateResult(t, tc.wantValidationResult, ignoreKeysMap)); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

// testParseValidateResult parse the validation result
// and return a map with parsed key value pairs.
// It will skip the keys from ignoreKeys.
func testParseValidateResult(tb testing.TB, s string, ignoreKeys map[string]struct{}) map[string]string {
	tb.Helper()

	vMap := map[string]string{}

	r := strings.Split(s, "\n")
	for _, value := range r {
		// Skip empty lines
		if len(value) == 0 {
			continue
		}

		k := strings.Split(value, "    ")
		// Handle headers like `----- Claims -----`
		if len(k) == 1 {
			vMap[k[0]] = ""
			continue
		}

		// Handle key value pairs
		if len(k) == 2 {
			// ignore keys we don't want to compare
			if _, ok := ignoreKeysMap[k[0]]; !ok {
				vMap[k[0]] = k[1]
			}
			continue
		}

		tb.Fatalf("Validation result isn't in correct format. Got validation result: %s", s)
	}
	return vMap
}
