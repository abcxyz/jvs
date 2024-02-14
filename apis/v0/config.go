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
	"errors"
	"fmt"
	"time"

	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v3"
)

// Config is the jvs client configuration.
type Config struct {
	// JWKSEndpoint is the full path (including protocol and port) to the JWKS
	// endpoint on a JVS server (e.g. https://jvs.corp:8080/.well-known/jwks).
	JWKSEndpoint string `yaml:"endpoint,omitempty" env:"ENDPOINT,overwrite"`

	// CacheTimeout is the duration that keys stay in cache before being revoked.
	CacheTimeout time.Duration `yaml:"cache_timeout" env:"CACHE_TIMEOUT,overwrite,default=5m"`

	// AllowBreakglass represents whether the jvs client allows breakglass.
	AllowBreakglass bool `yaml:"allow_breakglass" env:"ALLOW_BREAKGLASS,overwrite,default=false"`
}

// Validate checks if the config is valid.
func (cfg *Config) Validate() error {
	var merr error
	if cfg.JWKSEndpoint == "" {
		merr = errors.Join(merr, fmt.Errorf("endpoint must be set"))
	}
	if cfg.CacheTimeout <= 0 {
		merr = errors.Join(merr, fmt.Errorf("cache timeout must be a positive duration, got %q", cfg.CacheTimeout))
	}
	return merr
}

// LoadConfig calls the necessary methods to load in config using the OsLookuper
// which finds env variables specified on the host.
func LoadConfig(ctx context.Context, b []byte) (*Config, error) {
	return loadConfigFromLookuper(ctx, b, envconfig.OsLookuper())
}

// loadConfigFromLooker reads in a yaml file, applies ENV config overrides from
// the lookuper, and finally validates the config.
func loadConfigFromLookuper(ctx context.Context, b []byte, lookuper envconfig.Lookuper) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	// Process overrides from env vars.
	if err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:   &cfg,
		Lookuper: lookuper,
	}); err != nil {
		return nil, fmt.Errorf("failed to process environment variables: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed validating config: %w", err)
	}
	return &cfg, nil
}
