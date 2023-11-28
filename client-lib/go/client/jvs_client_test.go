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

package client

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/pkg/testutil"
)

func TestValidateJWT(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// create another key, to show the correct key is retrieved from cache and used for validation.
	privateKey2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// create another key that will NOT be registered as a JWK
	unregisteredKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	key := "projects/[PROJECT]/locations/[LOCATION]/keyRings/[KEY_RING]/cryptoKeys/[CRYPTO_KEY]"
	keyID := key + "/cryptoKeyVersions/[VERSION]-0"
	keyID2 := key + "/cryptoKeyVersions/[VERSION]-1"

	ecdsaKey, err := jwk.FromRaw(privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := ecdsaKey.Set(jwk.KeyIDKey, keyID); err != nil {
		t.Fatal(err)
	}
	ecdsaKey2, err := jwk.FromRaw(privateKey2.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := ecdsaKey2.Set(jwk.KeyIDKey, keyID2); err != nil {
		t.Fatal(err)
	}
	jwks := make(map[string][]jwk.Key)
	jwks["keys"] = []jwk.Key{ecdsaKey, ecdsaKey2}

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

	tok := testCreateToken(t, "test_id")
	tok2 := testCreateToken(t, "test_id_2")
	breakglassToken := testCreateBreakglassToken(t)

	tests := []struct {
		name            string
		jwt             string
		sub             string
		allowBreakglass bool
		wantErr         string
		wantToken       jwt.Token
	}{
		{
			name:      "happy_path",
			jwt:       testSignToken(t, tok, privateKey, keyID),
			sub:       "test_sub",
			wantToken: tok,
		},
		{
			name:      "other_key",
			jwt:       testSignToken(t, tok2, privateKey2, keyID2),
			sub:       "test_sub",
			wantToken: tok2,
		},
		{
			name:            "breakglass",
			jwt:             testSignBreakglassToken(t, breakglassToken),
			sub:             "test_sub",
			allowBreakglass: true,
			wantToken:       breakglassToken,
		},
		{
			name:            "forbid_breakglass",
			jwt:             testSignBreakglassToken(t, breakglassToken),
			sub:             "test_sub",
			allowBreakglass: false,
			wantErr:         "breakglass is forbidden, denying",
		},
		{
			name:    "invalid",
			jwt:     testSignToken(t, tok, unregisteredKey, keyID),
			sub:     "test_sub",
			wantErr: "failed to verify jwt",
		},
		{
			name:    "bad_subject",
			jwt:     testSignToken(t, tok, privateKey, keyID),
			sub:     "bad_sub",
			wantErr: "does not match expected subject",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client, err := NewJVSClient(ctx, &JVSConfig{
				JWKSEndpoint:    svr.URL + path,
				CacheTimeout:    5 * time.Minute,
				AllowBreakglass: tc.allowBreakglass,
			})
			if err != nil {
				t.Fatalf("failed to create JVS client: %v", err)
			}
			res, err := client.ValidateJWT(ctx, tc.jwt, tc.sub)

			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if err != nil {
				return
			}
			got, err := json.MarshalIndent(res, "", " ")
			if err != nil {
				t.Errorf("couldn't marshal returned token %v", err)
			}
			want, err := json.MarshalIndent(tc.wantToken, "", " ")
			if err != nil {
				t.Errorf("couldn't marshal expected token %v", err)
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Token diff (-want, +got): %v", diff)
			}
		})
	}
}

func testCreateToken(tb testing.TB, id string) jwt.Token {
	tb.Helper()

	tok, err := jwt.NewBuilder().
		Audience([]string{"test_aud"}).
		Expiration(time.Now().UTC().Add(5 * time.Minute)).
		JwtID(id).
		IssuedAt(time.Now().UTC()).
		Issuer(`test_iss`).
		NotBefore(time.Now().UTC()).
		Subject("test_sub").
		Build()
	if err != nil {
		tb.Fatalf("failed to build token: %s\n", err)
	}

	if err := jvspb.SetJustifications(tok, []*jvspb.Justification{
		{
			Category: "explanation",
			Value:    "this is a test explanation",
		},
	}); err != nil {
		tb.Fatal(err)
	}
	return tok
}

func testCreateBreakglassToken(tb testing.TB) jwt.Token {
	tb.Helper()

	tok := testCreateToken(tb, "breakglass")

	if err := jvspb.SetJustifications(tok, []*jvspb.Justification{
		{
			Category: "breakglass",
			Value:    "this is a breakglass token",
		},
	}); err != nil {
		tb.Fatal(err)
	}
	return tok
}

func testSignToken(tb testing.TB, tok jwt.Token, privateKey *ecdsa.PrivateKey, keyID string) string {
	tb.Helper()

	hdrs := jws.NewHeaders()
	if err := hdrs.Set(jws.KeyIDKey, keyID); err != nil {
		tb.Fatal(err)
	}

	valid, err := jwt.Sign(tok, jwt.WithKey(jwa.ES256, privateKey, jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		tb.Fatalf("failed to sign token: %s\n", err)
	}
	return string(valid)
}

func testSignBreakglassToken(tb testing.TB, token jwt.Token) string {
	tb.Helper()

	str, err := jvspb.CreateBreakglassToken(token, "testing")
	if err != nil {
		tb.Fatal(err)
	}
	return str
}
