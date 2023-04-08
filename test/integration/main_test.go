// Copyright 2023 The Authors (see AUTHORS file)
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

package integration

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Global integration test config.
var cfg *config

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

		log.Printf("main, cert rotator url: (%+v)", cfg.CertRotatorURL)
		log.Printf("main, api server: (%+v)", cfg.APISERVER)
		log.Printf("main, jwks endpoint: (%+v)", cfg.JwksEndpoint)
		log.Printf("main, api url: (%+v)", cfg.APIURL)
		log.Printf("main, jwks url: (%+v)", cfg.PublicKeyURL)
		log.Printf("main, ui url: (%+v)", cfg.UIURL)

		log.Printf("main, cert rotator auth: (%+v)", cfg.CertRotatorAuthToken)
		log.Printf("main, api auth: (%+v)", cfg.APIAuthToken)

		return m.Run()
	}())
}

// Test justification API and public key service.
func TestAPI(t *testing.T) {
	t.Parallel()

	t.Logf("api server: (%+v)", cfg.APISERVER)
	t.Logf("jwks endpoint: (%+v)", cfg.JwksEndpoint)
	cases := []struct {
		name    string
		args    []string
		wantOut string
		wantErr string
	}{
		{
			name: "happy_path",
			args: []string{
				"token",
				"-explanation=test",
				"-server", cfg.APISERVER,
				"-auth-token", cfg.APIAuthToken,
			},
			wantOut: `
----- Justifications -----
explanation    test
foo            bar

----- Claims -----
aud    [dev.abcxyz.jvs]
iat    1970-01-01 12:00AM UTC
iss    jvsctl
jti    test-jwt
nbf    1970-01-01 12:00AM UTC
sub    test-sub
`,
		},
		{
			name: "breakglass",
			args: []string{
				"token",
				"-explanation=prod is down",
				"-breakglass",
			},
			wantOut: `
Warning! This is a breakglass token.

----- Justifications -----
breakglass    prod is down

----- Claims -----
aud    [dev.abcxyz.jvs]
iat    1970-01-01 12:00AM UTC
iss    jvsctl
jti    test-jwt
nbf    1970-01-01 12:00AM UTC
sub    test-sub
`,
		},
		{
			name: "token_expired",
			args: []string{
				"token",
				"-explanation=test",
				"-ttl=1ns",
				"-server", cfg.APISERVER,
				"-auth-token", cfg.APIAuthToken,
			},
			wantErr: "error",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			token, err := exec.Command("jvsctl", tc.args...).Output() // #nosec G204
			if err != nil {
				t.Errorf("Process(%+v) failed to create token: %v", tc.name, err)
			}
			gotOut, err := exec.Command(
				"jvsctl", "validate",
				"-t", string(token),
				"-jwks_endpoint", cfg.JwksEndpoint,
			).Output() // #nosec G204
			if err != nil {
				t.Errorf("Process(%+v) failed to validate token: %v", tc.name, err)
			}
			if string(gotOut) != tc.wantOut {
				t.Errorf(
					"Process(%+v) want output (%+v) got output(%+v)",
					tc.name, tc.wantOut, gotOut)
			}
		})
	}
}

// Test cert rotator.
func TestCertRotator(t *testing.T) {
	t.Parallel()

	t.Logf("cert rotator url: (%+v)", cfg.CertRotatorURL)
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "GET", cfg.CertRotatorURL, nil)
	if err != nil {
		t.Fatalf("Failed to create cert rotator request: (%+v)", err)
	}

	req.Header.Set("Authorization", fmt.Sprint("Bearer ", cfg.CertRotatorAuthToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Test cert rotator got err: (%+v)", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf(
			"Rotate key got unexpected status: (%+v), with error: (%+v)",
			resp.StatusCode, resp.Body,
		)
	}
	defer resp.Body.Close()
}

// TODO(#158): add integration test for UI
