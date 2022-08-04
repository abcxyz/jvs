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
	"fmt"
	"time"

	"github.com/abcxyz/jvs/pkg/config"
	"github.com/hashicorp/go-multierror"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v2"
)

var versions = config.NewVersionList("1")

// JVSConfig is the jvs client configuration.
type JVSConfig struct {
	// Version is the version of the config.
	Version string `yaml:"version,omitempty" env:"VERSION,overwrite,default=1"`

	// JVS Endpoint. Expected to be fully qualified, including port. ex. http://127.0.0.1:8080
	JVSEndpoint string `yaml:"endpoint,omitempty" env:"ENDPOINT,overwrite"`

	// CacheTimeout is the duration that keys stay in cache before being revoked.
	CacheTimeout time.Duration `yaml:"cache_timeout" env:"CACHE_TIMEOUT,overwrite,default=5m"`

	// AllowBreakglass represents whether the jvs client supports breakglass.
	AllowBreakglass bool `yaml:"allow_breakglass" env:"ALLOW_BREAKGLASS,overwrite,default=true"`
}

// Validate checks if the config is valid.
func (cfg *JVSConfig) Validate() error {
	var err *multierror.Error
	if !versions.Contains(cfg.Version) {
		err = multierror.Append(err, fmt.Errorf("version %q is invalid, valid versions are: %q",
			cfg.Version, versions.List()))
	}
	if cfg.JVSEndpoint == "" {
		err = multierror.Append(err, fmt.Errorf("endpoint must be set"))
	}
	if cfg.CacheTimeout <= 0 {
		err = multierror.Append(err, fmt.Errorf("cache timeout must be a positive duration, got %q", cfg.CacheTimeout))
	}
	return err.ErrorOrNil()
}

// LoadJVSConfig calls the necessary methods to load in config using the OsLookuper which finds env variables specified on the host.
func LoadJVSConfig(ctx context.Context, b []byte) (*JVSConfig, error) {
	return loadJVSConfigFromLookuper(ctx, b, envconfig.OsLookuper())
}

// loadConfigFromLooker reads in a yaml file, applies ENV config overrides from the lookuper, and finally validates the config.
func loadJVSConfigFromLookuper(ctx context.Context, b []byte, lookuper envconfig.Lookuper) (*JVSConfig, error) {
	cfg := &JVSConfig{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, err
	}

	// Process overrides from env vars.
	l := envconfig.PrefixLookuper("JVS_", lookuper)
	if err := envconfig.ProcessWith(ctx, cfg, l); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed validating config: %w", err)
	}
	return cfg, nil
}
