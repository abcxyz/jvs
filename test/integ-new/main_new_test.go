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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	v0 "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/cli"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/testing/protocmp"
)

// Global integration test config.
var cfg *config

const (
	timestampFormat = time.RFC3339
	testTTLString   = "30m"
	testTTLTime     = 30 * time.Minute
)

type testClaim struct {
	Aud []string
	Iat string
	Exp string
	Iss string
	Req string
	Sub string
}

type testValidationResult struct {
	Breakglass     bool
	Justifications []v0.Justification
	Claims         testClaim
}

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
		name                     string
		isBreakglass             bool
		justification            string
		wantTestValidationResult *testValidationResult
	}{
		{
			name:          "none_breakglass",
			justification: "issues/12345",
			isBreakglass:  false,
			wantTestValidationResult: &testValidationResult{
				Breakglass: false,
				Justifications: []v0.Justification{
					{
						Category: "explanation",
						Value:    "issues/12345",
					},
				},
				Claims: testClaim{
					Aud: []string{"dev.abcxyz.jvs"},
					Exp: ts.Add(testTTLTime).UTC().Format(timestampFormat),
					Iat: ts.Format(timestampFormat),
					Iss: "jvs.abcxyz.dev",
					Req: cfg.ServiceAccount,
					Sub: cfg.ServiceAccount,
				},
			},
		},
		{
			name:          "breakglass",
			justification: "issues/12345",
			isBreakglass:  true,
			wantTestValidationResult: &testValidationResult{
				Breakglass: true,
				Justifications: []v0.Justification{
					{
						Category: "breakglass",
						Value:    "issues/12345",
					},
				},
				Claims: testClaim{
					Aud: []string{"dev.abcxyz.jvs"},
					Exp: ts.Add(testTTLTime).UTC().Format(timestampFormat),
					Iat: ts.Format(timestampFormat),
					Iss: "jvsctl",
				},
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
				t.Fatal(err)
			}

			token := stdout.String()

			validateTokenArgs := []string{"-token", token, "-jwks-endpoint", cfg.JWTEndpoint, "--format", "json"}

			var validateCmd cli.TokenValidateCommand
			_, stdout, _ = validateCmd.Pipe()

			if err := validateCmd.Run(ctx, validateTokenArgs); err != nil {
				t.Fatal(err)
			}

			gotTestValidationResult := &testValidationResult{}
			if err := json.Unmarshal(stdout.Bytes(), gotTestValidationResult); err != nil {
				t.Fatalf("failed to unmarshal validation result: %v", err)
			}

			cmpOpts := []cmp.Option{
				protocmp.Transform(),
				cmpopts.IgnoreFields(testClaim{}, "Exp", "Iat"),
			}

			if diff := cmp.Diff(tc.wantTestValidationResult, gotTestValidationResult, cmpOpts...); diff != "" {
				t.Errorf("wantValidationResult got unexpacted diff (-want, +got):\n%s", diff)
			}

			if err := testDiffClaimTimestamp(t, tc.wantTestValidationResult.Claims, gotTestValidationResult.Claims); err != nil {
				t.Errorf("Claims got unexpected diff: %v", err)
			}
		})
	}
}

// testDiffClaim compares the timestamp within the claims.
// The seconds in timestamp will be trimed before comparing,
// this is do help mitigate the delay when running tests.
func testDiffClaimTimestamp(tb testing.TB, want, got testClaim) error {
	tb.Helper()

	var rErr error

	if want.Exp[:len(want.Exp)-3] != got.Exp[:len(got.Exp)-3] {
		rErr = errors.Join(rErr, fmt.Errorf(fmt.Sprintf("got unexpected exp timestamp (-want, +got)\n -%s\n +%s\n", want.Exp, got.Exp)))
	}

	if want.Exp[:len(want.Iat)-3] != got.Exp[:len(got.Iat)-3] {
		rErr = errors.Join(rErr, fmt.Errorf(fmt.Sprintf("got unexpected iat timestamp (-want, +got)\n -%s\n +%s\n", want.Iat, got.Iat)))
	}

	return rErr
}
