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

// Package config provides configuration-related files and methods.
package config

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v2"
)

// JustificationConfig is the full jvs config.
type JustificationConfig struct {
	// Service configuration.
	Port uint16 `yaml:"port,omitempty" env:"VERSION,overwrite"`
}

// Validate checks if the config is valid.
func (cfg *JustificationConfig) Validate() error {
	cfg.SetDefault()

	var err error
	if cfg.Port <= 0 {
		err = multierror.Append(err, fmt.Errorf("port is invalid: %v", cfg.Port))
	}

	return err
}

// SetDefault sets default for the config.
func (cfg *JustificationConfig) SetDefault() {
	// TODO: set defaults for other fields if necessary.
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
}

// LoadJustificationConfig calls the necessary methods to load in config using the OsLookuper which finds env variables specified on the host.
func LoadJustificationConfig(ctx context.Context, b []byte) (*JustificationConfig, error) {
	return loadJustificationConfigFromLookuper(ctx, b, envconfig.OsLookuper())
}

// loadConfigFromLooker reads in a yaml file, applies ENV config overrides from the lookuper, and finally validates the config.
func loadJustificationConfigFromLookuper(ctx context.Context, b []byte, lookuper envconfig.Lookuper) (*JustificationConfig, error) {
	cfg := &JustificationConfig{}
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
