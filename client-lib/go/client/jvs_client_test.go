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
	"strings"
	"testing"
	"time"

	v0 "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
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

	key := "projects/[PROJECT]/locations/[LOCATION]/keyRings/[KEY_RING]/cryptoKeys/[CRYPTO_KEY]"
	keyID := key + "/cryptoKeyVersions/[VERSION]-0"
	keyID2 := key + "/cryptoKeyVersions/[VERSION]-1"

	ecdsaKey, err := jwk.FromRaw(privateKey.PublicKey)
	ecdsaKey.Set(jwk.KeyIDKey, keyID)
	ecdsaKey2, err := jwk.FromRaw(privateKey2.PublicKey)
	ecdsaKey2.Set(jwk.KeyIDKey, keyID2)
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
		fmt.Fprintf(w, string(j))
	})

	svr := httptest.NewServer(mux)

	t.Cleanup(func() {
		svr.Close()
	})

	client, err := NewJVSClient(ctx, &JVSConfig{
		Version:      1,
		JVSEndpoint:  svr.URL + path,
		CacheTimeout: 5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("failed to create JVS client: %v", err)
	}

	tok, err := jwt.NewBuilder().
		Audience([]string{"test_aud"}).
		Expiration(time.Now().Add(5 * time.Minute)).
		JwtID("test_id").
		IssuedAt(time.Now()).
		Issuer(`test_iss`).
		NotBefore(time.Now()).
		Subject("test_sub").
		Build()
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}
	tok.Set("justs", []*v0.Justification{
		{
			Category: "explanation",
			Value:    "this is a test explanation",
		},
	})
	hdrs := jws.NewHeaders()
	hdrs.Set(jws.KeyIDKey, keyID)

	valid, err := jwt.Sign(tok, jwt.WithKey(jwa.ES256, privateKey, jws.WithProtectedHeaders(hdrs)))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	validJWT := string(valid)

	// Create and sign a token with 2nd key
	tok2, err := jwt.NewBuilder().
		Audience([]string{"test_aud"}).
		Expiration(time.Now().Add(5 * time.Minute)).
		JwtID("test_id_2").
		IssuedAt(time.Now()).
		Issuer(`test_iss`).
		NotBefore(time.Now()).
		Subject("test_sub").
		Build()
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}
	tok2.Set("justs", []*v0.Justification{
		{
			Category: "explanation",
			Value:    "this is a test explanation",
		},
	})
	hdrs2 := jws.NewHeaders()
	hdrs2.Set(jws.KeyIDKey, keyID2)

	valid2, err := jwt.Sign(tok2, jwt.WithKey(jwa.ES256, privateKey2, jws.WithProtectedHeaders(hdrs2)))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	validJWT2 := string(valid2)

	unsig, err := jwt.NewSerializer().Serialize(tok)
	if err != nil {
		t.Fatal("Couldn't get signing string.")
	}
	unsignedJWT := string(unsig)

	split := strings.Split(validJWT2, ".")
	sig := split[len(split)-1]

	invalidSignatureJWT := unsignedJWT + sig // signature from a different JWT

	tests := []struct {
		name      string
		jwt       string
		wantErr   string
		wantToken jwt.Token
	}{
		{
			name:      "happy-path",
			jwt:       validJWT,
			wantToken: tok,
		}, {
			name:      "other-key",
			jwt:       validJWT2,
			wantToken: tok2,
		}, {
			name:    "unsigned",
			jwt:     unsignedJWT,
			wantErr: "required field \"signatures\" not present",
		}, {
			name:    "invalid",
			jwt:     invalidSignatureJWT,
			wantErr: "failed to verify jwt",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			res, err := client.ValidateJWT(ctx, tc.jwt)
			testutil.ErrCmp(t, tc.wantErr, err)
			if err != nil {
				return
			}
			got, err := json.MarshalIndent(res, "", " ")
			if err != nil {
				t.Error(fmt.Errorf("couldn't marshall returned token %w", err))
			}
			want, err := json.MarshalIndent(tc.wantToken, "", " ")
			if err != nil {
				t.Error(fmt.Errorf("couldn't marshall expected token %w", err))
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("Token diff (-want, +got): %v", diff)
			}
		})
	}
}
