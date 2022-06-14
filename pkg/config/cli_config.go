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

package config

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultCLIConfigVersion is the default CLI config version.
	DefaultCLIConfigVersion = 1
)

type CLIConfig struct {
	// Version is the version of the config.
	Version uint8 `yaml:"version,omitempty" env:"VERSION,overwrite"`

	// Server is the JVS server address.
	Server string `yaml:"server,omitempty" env:"SERVER,overwrite"`
}

// Validate checks if the config is valid.
func (cfg *CLIConfig) Validate() error {
	cfg.SetDefault()

	var err *multierror.Error
	if cfg.Server == "" {
		err = multierror.Append(err, fmt.Errorf("missing JVS server address"))
	}

	return err.ErrorOrNil()
}

// SetDefault sets default for the config.
func (cfg *CLIConfig) SetDefault() {
	if cfg.Version == 0 {
		cfg.Version = DefaultCLIConfigVersion
	}
}

// LoadConfig calls the necessary methods to load in config using the OsLookuper which finds env variables specified on the host.
func LoadCLIConfig(ctx context.Context, b []byte) (*CLIConfig, error) {
	return loadCLIConfigFromLookuper(ctx, b, envconfig.OsLookuper())
}

// loadConfigFromLooker reads in a yaml file, applies ENV config overrides from the lookuper, and finally validates the config.
func loadCLIConfigFromLookuper(ctx context.Context, b []byte, lookuper envconfig.Lookuper) (*CLIConfig, error) {
	cfg := &CLIConfig{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// Process overrides from env vars.
	l := envconfig.PrefixLookuper("JVSCTL_", lookuper)
	if err := envconfig.ProcessWith(ctx, cfg, l); err != nil {
		return nil, fmt.Errorf("failed to process environment: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed validating config: %w", err)
	}

	return cfg, nil
}
