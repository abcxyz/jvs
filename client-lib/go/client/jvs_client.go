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

// Package client provides a client library for getting JWK keys from JVS and then
package client

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"

	"github.com/abcxyz/jvs/pkg/cache"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/golang-jwt/jwt"
)

type JVSClient struct {
	Config *JVSConfig
	Cache  *cache.Cache[*ecdsa.PublicKey]
	mu     sync.RWMutex
}

// NewJVSClient returns a JVSClient with the cache initialized
func NewJVSClient(config *JVSConfig) *JVSClient {
	return &JVSClient{
		Config: config,
		Cache:  cache.New[*ecdsa.PublicKey](config.CacheTimeout),
	}
}

// ValidateJWT takes a jwt string, converts it to a JWT, and validates the signature.
func (j *JVSClient) ValidateJWT(jwtStr string) (*jwt.Token, error) {
	parts := strings.Split(jwtStr, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid jwt string %s", jwtStr)
	}

	// We first have to deserialize the token without any validation in order to get the key id.
	token, _ := jwt.Parse(jwtStr, nil) // parse without validation
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("unrecognized claims format %v", claims)
	}
	keyID, ok := claims["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("unrecognized key id format. %s", keyID)
	}

	// Now that we have the keyID used to sign the token, get the key.
	pubKey, err := j.getFromCache(keyID)
	if err != nil {
		return nil, err
	}

	// verify the jwt using the key
	if err := jwt.SigningMethodES256.Verify(strings.Join(parts[0:2], "."), parts[2], pubKey); err != nil {
		return nil, fmt.Errorf("unable to verify signed jwt string. %w", err)
	}
	return token, nil
}

// getFromCache attemps to get a key from the cache. On a failure, the cache is updated.
// This is surrounded by a lock in order to ensure that only 1 update/get operation occurs at a time.
// This works around a limitation in the cache implementation that doesn't allow for direct setting of keys
// when using write-through, and on a cache miss we want to update all keys.
func (j *JVSClient) getFromCache(key string) (*ecdsa.PublicKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	val, ok := j.Cache.Lookup(key)
	if !ok {
		err := j.updateCache()
		if err != nil {
			return nil, fmt.Errorf("unable to update cache %w", err)
		}
		val, ok = j.Cache.Lookup(key)
		if !ok {
			return nil, fmt.Errorf("unable to get public key for key id %s", key)
		}
	}
	return val, nil
}

// refresh our list of JWKs from the jvs endpoint
func (j *JVSClient) updateCache() error {
	resp, err := http.Get(j.Config.JVSEndpoint + "/.well-known/jwks")
	if err != nil {
		return fmt.Errorf("err while calling JVS public key endpoint %w", err)
	}
	var jwks jvscrypto.JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("err while decoding JVS response %w", err)
	}

	for _, token := range jwks.Keys {
		x, err := decode(token.X)
		if err != nil {
			return err
		}
		y, err := decode(token.Y)
		if err != nil {
			return err
		}
		key := &ecdsa.PublicKey{
			Curve: elliptic.P256(), // assumption, should use the one from the json
			X:     x,
			Y:     y,
		}
		if err := j.Cache.Set(token.ID, key); err != nil {
			return err
		}
	}

	return nil
}

func decode(val string) (*big.Int, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(val)
	i := new(big.Int)
	if err != nil {
		return i, fmt.Errorf("err while decoding token %w", err)
	}
	i.SetBytes(bytes)
	return i, nil
}
