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

package v0

import (
	"context"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Client allows for getting JWK keys from the JVS and validating JWTs with
// those keys.
type Client struct {
	config *Config
	keys   jwk.Set
}

// NewClient returns a JVSClient with the cache initialized.
func NewClient(ctx context.Context, config *Config) (*Client, error) {
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

	return &Client{
		config: config,
		keys:   cached,
	}, nil
}

// ValidateJWT takes a jwt string, converts it to a JWT, and validates the
// signature against the keys in the JWKs endpoint.
func (j *Client) ValidateJWT(ctx context.Context, jwtStr, expectedSubject string) (jwt.Token, error) {
	// Handle breakglass tokens
	token, err := ParseBreakglassToken(ctx, jwtStr)
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
	token, err = jwt.Parse([]byte(jwtStr),
		jwt.WithContext(ctx),
		jwt.WithKeySet(j.keys, jws.WithInferAlgorithmFromKey(true)),
		jwt.WithAcceptableSkew(5*time.Second),
		WithTypedJustifications(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify jwt: %w", err)
	}

	if got, want := token.Subject(), expectedSubject; got != want && expectedSubject != "" {
		return nil, fmt.Errorf("subject %q does not match expected subject %q", got, want)
	}

	return token, nil
}
