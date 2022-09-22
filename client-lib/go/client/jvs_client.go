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

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// JVSClient allows for getting JWK keys from the JVS and validating JWTs with
// those keys.
type JVSClient struct {
	config *JVSConfig
	keys   jwk.Set
}

// NewJVSClient returns a JVSClient with the cache initialized.
func NewJVSClient(ctx context.Context, config *JVSConfig) (*JVSClient, error) {
	c := jwk.NewCache(ctx)
	if err := c.Register(config.JWKSEndpoint, jwk.WithMinRefreshInterval(config.CacheTimeout)); err != nil {
		return nil, fmt.Errorf("failed to register: %w", err)
	}

	// check that cache is correctly set up and certs are available
	if _, err := c.Refresh(ctx, config.JWKSEndpoint); err != nil {
		return nil, fmt.Errorf("failed to retrieve JVS public keys: %w", err)
	}

	cached := jwk.NewCachedSet(c, config.JWKSEndpoint)

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate configuration: %w", err)
	}

	return &JVSClient{
		config: config,
		keys:   cached,
	}, nil
}

// ValidateJWT takes a jwt string, converts it to a JWT, and validates the
// signature against the keys in the JWKs endpoint.
func (j *JVSClient) ValidateJWT(jwtStr string) (jwt.Token, error) {
	// Handle breakglass tokens
	token, err := jvspb.ParseBreakglassToken(jwtStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse breakglass token: %w", err)
	}
	if token != nil {
		if !j.config.AllowBreakglass {
			return nil, fmt.Errorf("breakglass is forbidden, denying")
		}
		return token, nil
	}

	// If we got this far, the token was not breakglass, so parse as normal.
	token, err = jwt.ParseString(jwtStr,
		jvspb.WithTypedJustifications(),
		jwt.WithKeySet(j.keys, jws.WithInferAlgorithmFromKey(true)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify jwt: %w", err)
	}
	return token, nil
}
