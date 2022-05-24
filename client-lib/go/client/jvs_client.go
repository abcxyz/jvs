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

// Package client provides a client library for JVS
package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// JVSClient allows for getting JWK keys from the JVS and validating JWTs with those keys
type JVSClient struct {
	config *JVSConfig
	keys   jwk.Set
	mu     sync.RWMutex
}

// NewJVSClient returns a JVSClient with the cache initialized
func NewJVSClient(ctx context.Context, config *JVSConfig) (*JVSClient, error) {
	c := jwk.NewCache(ctx)
	c.Register(config.JVSEndpoint, jwk.WithMinRefreshInterval(config.CacheTimeout))

	// check that cache is correctly set up and certs are available
	_, err := c.Refresh(ctx, config.JVSEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve JVS public keys: %w", err)
	}

	cached := jwk.NewCachedSet(c, config.JVSEndpoint)

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate configuration: %w", err)
	}

	return &JVSClient{
		config: config,
		keys:   cached,
	}, nil
}

// ValidateJWT takes a jwt string, converts it to a JWT, and validates the signature.
func (j *JVSClient) ValidateJWT(ctx context.Context, jwtStr string) (*jwt.Token, error) {
	verifiedToken, err := jwt.Parse([]byte(jwtStr), jwt.WithKeySet(j.keys, jws.WithInferAlgorithmFromKey(true)))
	if err != nil {
		return nil, fmt.Errorf("failed to verify jwt %s: %w", jwtStr, err)
	}

	return &verifiedToken, nil
}
