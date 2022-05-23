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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
)

func TestValidateJWT(t *testing.T) {
	t.Parallel()

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
	keyID := jvscrypto.KeyID(key + "/cryptoKeyVersions/[VERSION]-0")
	keyID2 := jvscrypto.KeyID(key + "/cryptoKeyVersions/[VERSION]-1")

	ecdsaKey := &jvscrypto.ECDSAKey{
		Curve: "P-256",
		ID:    keyID,
		Type:  "EC",
		X:     base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.X.Bytes()),
		Y:     base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.Y.Bytes()),
	}

	ecdsaKey2 := &jvscrypto.ECDSAKey{
		Curve: "P-256",
		ID:    keyID2,
		Type:  "EC",
		X:     base64.RawURLEncoding.EncodeToString(privateKey2.PublicKey.X.Bytes()),
		Y:     base64.RawURLEncoding.EncodeToString(privateKey2.PublicKey.Y.Bytes()),
	}

	jwks := &jvscrypto.JWKS{
		Keys: []*jvscrypto.ECDSAKey{
			ecdsaKey,
			ecdsaKey2,
		},
	}

	json, err := json.Marshal(jwks)
	if err != nil {
		t.Fatal("couldn't create jwks json")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, string(json))
	})

	svr := httptest.NewServer(mux)

	t.Cleanup(func() {
		svr.Close()
	})

	client := NewJVSClient(&JVSConfig{
		Version:      1,
		JVSEndpoint:  svr.URL,
		CacheTimeout: 1 * time.Minute,
	})

	claims := &jvspb.JVSClaims{
		StandardClaims: &jwt.StandardClaims{
			Audience:  "test_aud",
			ExpiresAt: 100,
			Id:        uuid.New().String(),
			IssuedAt:  10,
			Issuer:    "test_iss",
			NotBefore: 10,
			Subject:   "test_sub",
		},
		KeyID: keyID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	validJWT, err := jvscrypto.SignToken(token, privateKey)
	if err != nil {
		t.Fatal("Couldn't sign token.")
	}

	claims2 := &jvspb.JVSClaims{
		StandardClaims: &jwt.StandardClaims{
			Audience:  "test_aud",
			ExpiresAt: 100,
			Id:        uuid.New().String(),
			IssuedAt:  10,
			Issuer:    "test_iss",
			NotBefore: 10,
			Subject:   "test_sub",
		},
		KeyID: keyID2,
	}
	token2 := jwt.NewWithClaims(jwt.SigningMethodES256, claims2)

	validJWT2, err := jvscrypto.SignToken(token2, privateKey2)
	if err != nil {
		t.Fatal("Couldn't sign token.")
	}

	unsignedJWT, err := token.SigningString()
	if err != nil {
		t.Fatal("Couldn't get signing string.")
	}

	invalidSignatureJWT := unsignedJWT + ".SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c" // signature from a different JWT

	tests := []struct {
		name    string
		jwt     string
		wantErr string
	}{
		{
			name: "happy-path",
			jwt:  validJWT,
		}, {
			name: "other-key",
			jwt:  validJWT2,
		}, {
			name:    "unsigned",
			jwt:     unsignedJWT,
			wantErr: "invalid jwt string",
		}, {
			name:    "invalid",
			jwt:     invalidSignatureJWT,
			wantErr: "unable to verify signed jwt string",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := client.ValidateJWT(tc.jwt)
			testutil.ErrCmp(t, tc.wantErr, err)
		})
	}
}
