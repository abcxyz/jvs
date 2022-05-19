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

	"github.com/hashicorp/go-multierror"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v2"
)

const (
	// Version default for config.
	Version = 1
)

// JVSConfig is the full jvs config.
type JVSConfig struct {
	// Version is the version of the config.
	Version uint8 `yaml:"version,omitempty" env:"VERSION,overwrite"`

	// Service configuration.
	JVSEndpoint string `yaml:"endpoint,omitempty" env:"ENDPOINT,overwrite"`

	CacheTimeout time.Duration `yaml:"cache_timeout" env:"CACHE_TIMEOUT,overwrite"`
}

// Validate checks if the config is valid.
func (cfg *JVSConfig) Validate() error {
	cfg.SetDefault()
	var err *multierror.Error
	if cfg.Version != Version {
		err = multierror.Append(err, fmt.Errorf("unexpected Version %d want %d", cfg.Version, Version))
	}
	if cfg.JVSEndpoint == "" {
		err = multierror.Append(err, fmt.Errorf("endpoint must be set"))
	}
	return err.ErrorOrNil()
}

// SetDefault sets default for the config.
func (cfg *JVSConfig) SetDefault() {
	if cfg.Version == 0 {
		cfg.Version = Version
	}
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

	if cfg.CacheTimeout == 0 {
		// env config lib doesn't gracefully handle env overrides with defaults, have to set manually.
		cfg.CacheTimeout = 5 * time.Minute
	}

	return cfg, nil
}
