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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
		t.Fatalf("failed to set justifications in token: %v", err)
	}
	signedToken := testSignToken(t, token, privateKey, keyID)
	breakglassToken, err := jvspb.CreateBreakglassToken(token, "prod is down")
	if err != nil {
		t.Fatalf("failed to build breakglass token: %v", err)
	}

	cases := []struct {
		name   string
		config *config.CLIConfig
		args   []string
		stdin  io.Reader
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
			expErr: `token is required`,
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
			expOut: `
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
			config: &config.CLIConfig{
				JWKSEndpoint: svr.URL + path,
			},
			args: []string{"-t", breakglassToken},
			expOut: `
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
			name: "with_subject_good",
			config: &config.CLIConfig{
				JWKSEndpoint: svr.URL + path,
			},
			args: []string{
				"-t", signedToken,
				"-s", "test-sub",
			},
			expOut: `
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
			name: "with_subject_bad",
			config: &config.CLIConfig{
				JWKSEndpoint: svr.URL + path,
			},
			args: []string{
				"-t", signedToken,
				"-s", "bad-sub",
			},
			expErr: `does not match expected subject`,
		},
		{
			name: "from_stdin",
			config: &config.CLIConfig{
				JWKSEndpoint: svr.URL + path,
			},
			args:  []string{"-t", "-"},
			stdin: strings.NewReader(signedToken),
			expOut: `
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
			name: "json",
			config: &config.CLIConfig{
				JWKSEndpoint: svr.URL + path,
			},
			args:   []string{"-t", breakglassToken, "-f", "json"},
			expOut: `{"breakglass":true,"justifications":[{"category":"breakglass","value":"prod is down"}],"claims":{"aud":["dev.abcxyz.jvs"],"iat":"1970-01-01T00:00:00Z","iss":"jvsctl","jti":"test-jwt","nbf":"1970-01-01T00:00:00Z","sub":"test-sub"}}`,
		},
		{
			name: "yaml",
			config: &config.CLIConfig{
				JWKSEndpoint: svr.URL + path,
			},
			args: []string{"-t", signedToken, "-f", "yaml"},
			expOut: `
breakglass: false
justifications:
  - category: explanation
    value: test
  - category: foo
    value: bar
claims:
  aud:
    - dev.abcxyz.jvs
  iat: 1970-01-01T00:00:00Z
  iss: jvsctl
  jti: test-jwt
  nbf: 1970-01-01T00:00:00Z
  sub: test-sub
`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cmd := newValidateCmd(tc.config)
			stdout, _, gotErr := testExecuteCommandStdin(t, cmd, tc.stdin, tc.args...)
			if diff := testutil.DiffErrString(gotErr, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if gotErr != nil {
				return
			}

			if diff := cmp.Diff(strings.TrimSpace(tc.expOut), strings.TrimSpace(stdout)); diff != "" {
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
		Subject("test-sub").
		Build()
	if err != nil {
		tb.Fatalf("failed to create unsigned token: %v", err)
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
		tb.Fatalf("failed to sign token: %v", err)
	}
	return string(token)
}
