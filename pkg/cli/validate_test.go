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

package cli

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestNewValidateCmd(t *testing.T) {
	t.Parallel()

	// Setup jwks server
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	keyID := "test_key_id"
	ecdsaKey, err := jwk.FromRaw(privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := ecdsaKey.Set(jwk.KeyIDKey, keyID); err != nil {
		t.Fatal(err)
	}
	jwks := make(map[string][]jwk.Key)
	jwks["keys"] = []jwk.Key{ecdsaKey}

	j, err := json.MarshalIndent(jwks, "", " ")
	if err != nil {
		t.Fatal("couldn't create jwks json")
	}
	path := "/.well-known/jwks"
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", j)
	})

	svr := httptest.NewServer(mux)

	t.Cleanup(func() {
		svr.Close()
	})

	// Build test tokens
	token := testTokenBuilder(t)
	if err := jvspb.SetJustifications(token, []*jvspb.Justification{
		{
			Category: "explanation",
			Value:    "test",
		},
		{
			Category: "foo",
			Value:    "bar",
		},
	}); err != nil {
		t.Fatalf("failed to set justifications in token: %s", err)
	}
	signedToken := testSignToken(t, token, privateKey, keyID)
	breakglassToken, err := jvspb.CreateBreakglassToken(token, "prod is down")
	if err != nil {
		t.Fatalf("failed to build breakglass token: %s", err)
	}

	cases := []struct {
		name   string
		config *config.CLIConfig
		args   []string
		pipe   bool
		expOut string
		expErr string
	}{
		{
			name:   "too_many_args",
			args:   []string{"foo"},
			expErr: `accepts 0 arg(s)`,
		},
		{
			name:   "missing_token",
			args:   nil,
			expErr: `"token" not set`,
		},
		{
			name: "invalid_token",
			config: &config.CLIConfig{
				JWKSEndpoint: svr.URL + path,
			},
			args:   []string{"-t=invalid token"},
			expErr: `failed to parse token headers`,
		},
		{
			name: "signed",
			config: &config.CLIConfig{
				JWKSEndpoint: svr.URL + path,
			},
			args: []string{"-t", signedToken},
			expOut: `-----BREAKGLASS-----
false

----JUSTIFICATION----
explanation  "test"
foo          "bar"

---STANDARD CLAIMS---
aud  ["dev.abcxyz.jvs"]
iat  "1970-01-01T00:00:00Z"
iss  "jvsctl"
jti  "test-jwt"
nbf  "1970-01-01T00:00:00Z"
sub  "jvsctl"
`,
		},
		{
			name: "breakglass",
			config: &config.CLIConfig{
				JWKSEndpoint: svr.URL + path,
			},
			args: []string{"-t", breakglassToken},
			expOut: `-----BREAKGLASS-----
true

----JUSTIFICATION----
breakglass  "prod is down"

---STANDARD CLAIMS---
aud  ["dev.abcxyz.jvs"]
iat  "1970-01-01T00:00:00Z"
iss  "jvsctl"
jti  "test-jwt"
nbf  "1970-01-01T00:00:00Z"
sub  "jvsctl"
`,
		},
		{
			name: "signed_from_pipe",
			config: &config.CLIConfig{
				JWKSEndpoint: svr.URL + path,
			},
			args: []string{"-t", "-"},
			pipe: true,
			expOut: `-----BREAKGLASS-----
false

----JUSTIFICATION----
explanation  "test"
foo          "bar"

---STANDARD CLAIMS---
aud  ["dev.abcxyz.jvs"]
iat  "1970-01-01T00:00:00Z"
iss  "jvsctl"
jti  "test-jwt"
nbf  "1970-01-01T00:00:00Z"
sub  "jvsctl"
`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cmd := newValidateCmd(tc.config)
			var stdout string
			var gotErr error
			if tc.pipe {
				// create tempFile with token
				tempFile, err := os.CreateTemp("", "validate_test*")
				if err != nil {
					t.Errorf("failed to create temp file: %s", err)
				}
				// remove tempFile
				defer os.Remove(tempFile.Name())

				if _, err := tempFile.Write([]byte(signedToken)); err != nil {
					t.Errorf("failed to write to temp file: %s", err)
				}
				if _, err := tempFile.Seek(0, 0); err != nil {
					t.Errorf("failed to seek to the beginning of the temp file: %s", err)
				}

				stdout, _, gotErr = testExecuteCommandStdin(t, cmd, tempFile, tc.args...)
				if err := tempFile.Close(); err != nil {
					t.Errorf("failed to close temp file: %s", err)
				}
			} else {
				stdout, _, gotErr = testExecuteCommand(t, cmd, tc.args...)
			}
			if diff := testutil.DiffErrString(gotErr, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if gotErr != nil {
				return
			}

			if diff := cmp.Diff(tc.expOut, stdout); diff != "" {
				t.Errorf("output: diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func testTokenBuilder(tb testing.TB) jwt.Token {
	tb.Helper()

	now := time.Unix(0, 0).UTC()
	token, err := jwt.NewBuilder().
		Audience([]string{justification.DefaultAudience}).
		IssuedAt(now).
		Issuer(Issuer).
		JwtID("test-jwt").
		NotBefore(now).
		Subject(Subject).
		Build()
	if err != nil {
		tb.Fatalf("failed to create unsigned token: %s", err)
	}
	return token
}

func testSignToken(tb testing.TB, unsignedToken jwt.Token, privateKey *ecdsa.PrivateKey, keyID string) string {
	tb.Helper()

	headers := jws.NewHeaders()
	if err := headers.Set(jws.KeyIDKey, keyID); err != nil {
		tb.Fatal(err)
	}

	token, err := jwt.Sign(unsignedToken, jwt.WithKey(jwa.ES256, privateKey, jws.WithProtectedHeaders(headers)))
	if err != nil {
		tb.Fatalf("failed to sign token: %s", err)
	}
	return string(token)
}
