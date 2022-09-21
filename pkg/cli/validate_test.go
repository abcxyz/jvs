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
	"strings"
	"testing"

	v0 "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/pkg/testutil"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/spf13/cobra"
)

func TestRunValidateCmd(t *testing.T) {
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

	breakglassToken, err := v0.CreateBreakglassToken(testTokenBuilder(t), "testing")
	if err != nil {
		t.Fatalf("failed to build breakglass token: %s", err)
	}

	cfg = &config.CLIConfig{
		JWKSEndpoint: svr.URL + path,
	}

	tests := []struct {
		name            string
		token           string
		wantOut         string
		wantErr         string
		allowBreakglass bool
	}{
		{
			name:    "signed",
			token:   testSignToken(t, testTokenBuilder(t), privateKey, keyID),
			wantOut: "Token is valid",
		},
		{
			name:            "breakglass",
			token:           breakglassToken,
			wantOut:         "Token is valid",
			allowBreakglass: true,
		},
		{
			name:            "breakglass_not_allowed",
			token:           breakglassToken,
			wantErr:         "breakglass is forbidden, denying",
			allowBreakglass: false,
		},
		{
			name:    "invalid",
			token:   "token",
			wantErr: "failed to parse token headers",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flagToken = tc.token
			flagAllowBreakglass = tc.allowBreakglass
			t.Cleanup(testRunValidateCmdCleanup)

			buf := &strings.Builder{}
			cmd := &cobra.Command{}
			cmd.SetOut(buf)

			err := runValidateCmd(cmd, nil)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
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
	flagAllowBreakglass = false
}

func testTokenBuilder(tb testing.TB) jwt.Token {
	tb.Helper()

	token, err := jwt.NewBuilder().Build()
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
