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
	"strings"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	UnsignedPostfix    = ".NOT_SIGNED"
	BreakglassCategory = "breakglass"
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
	// Handle unsigned tokens.
	if strings.HasSuffix(jwtStr, UnsignedPostfix) {
		token, err := jwt.Parse([]byte(jwtStr),
			jvspb.WithTypedJustifications(),
			jwt.WithVerify(false))
		if err != nil {
			return nil, fmt.Errorf("failed to parse jwt %s: %w", jwtStr, err)
		}
		if err := j.unsignedTokenValidAndAllowed(token); err != nil {
			return nil, fmt.Errorf("token unsigned and could not be validated: %w", err)
		}
		return token, nil
	}

	token, err := jwt.ParseString(jwtStr,
		jvspb.WithTypedJustifications(),
		jwt.WithKeySet(j.keys, jws.WithInferAlgorithmFromKey(true)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify jwt: %w", err)
	}
	return token, nil
}

func (j *JVSClient) unsignedTokenValidAndAllowed(token jwt.Token) error {
	if !j.config.AllowBreakglass {
		return fmt.Errorf("breakglass is forbidden, denying")
	}

	justifications, err := jvspb.GetJustifications(token)
	if err != nil {
		return err
	}

	for _, justification := range justifications {
		if justification.GetCategory() == BreakglassCategory {
			return nil
		}
	}
	return fmt.Errorf("justification category is not breakglass, denying")
}
