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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/pkg/testutil"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/spf13/cobra"
)

func TestRunValidateCmd(t *testing.T) {
	// Cannot parallel because of the global CLI config.
	// setup jwks server
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

	breakglassToken, err := jvspb.CreateBreakglassToken(testTokenBuilder(t), "testing")
	if err != nil {
		t.Fatalf("failed to build breakglass token: %s", err)
	}

	cfg = &config.CLIConfig{
		JWKSEndpoint: svr.URL + path,
	}

	tests := []struct {
		name    string
		token   string
		pipe    bool
		wantOut string
		wantErr string
	}{
		{
			name:  "signed",
			token: testSignToken(t, testTokenBuilder(t), privateKey, keyID),
			wantOut: `breakglass false
jti        "test_id"
valid      true
`,
		},
		{
			name:  "breakglass",
			token: breakglassToken,
			wantOut: `breakglass true
jti        "test_id"
justs      [{"category":"breakglass","value":"testing"}]
valid      true
`,
		},
		{
			name:  "signed_from_pipe",
			token: testSignToken(t, testTokenBuilder(t), privateKey, keyID),
			pipe:  true,
			wantOut: `breakglass false
jti        "test_id"
valid      true
`,
		},
		{
			name:    "invalid",
			token:   "token",
			wantErr: "failed to parse token headers",
		},
	}

	for _, tc := range tests {
		// Cannot parallel because of the global CLI config.
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(testRunValidateCmdCleanup)

			buf := &strings.Builder{}
			cmd := &cobra.Command{}
			cmd.SetOut(buf)
			var cmdErr error = nil

			if tc.pipe {
				// create tempFile with token
				tempFile, err := ioutil.TempFile("", "validate_test*")
				if err != nil {
					t.Errorf("failed to create temp file: %s", err)
				}
				// remove tempFile
				defer os.Remove(tempFile.Name())

				if _, err := tempFile.Write([]byte(tc.token)); err != nil {
					t.Errorf("failed to write to temp file: %s", err)
				}
				if _, err := tempFile.Seek(0, 0); err != nil {
					t.Errorf("failed to seek to the beginning of the temp file: %s", err)
				}

				// swap Stdin with tempFile
				oldStdin := os.Stdin
				// restore Stdin
				defer func() { os.Stdin = oldStdin }()
				os.Stdin = tempFile

				flagToken = "-"
				cmdErr = runValidateCmd(cmd, nil)
				if err := tempFile.Close(); err != nil {
					t.Errorf("failed to close temp file: %s", err)
				}
			} else {
				flagToken = tc.token
				cmdErr = runValidateCmd(cmd, nil)
			}
			if diff := testutil.DiffErrString(cmdErr, tc.wantErr); diff != "" {
				t.Errorf("unexpected err: %s", diff)
			}
			if gotOut := buf.String(); gotOut != tc.wantOut {
				t.Errorf("out got=%q, want=%q", gotOut, tc.wantOut)
			}
		})
	}
}

func testRunValidateCmdCleanup() {
	flagToken = ""
}

func testTokenBuilder(tb testing.TB) jwt.Token {
	tb.Helper()

	token, err := jwt.NewBuilder().JwtID("test_id").Build()
	if err != nil {
		tb.Fatalf("failed to build unsigned token: %s", err)
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
