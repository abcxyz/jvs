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
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v2"
)

const (
	// Version default for config.
	Version = "v1alpha1"
)

// CryptoConfig is the full jvs config.
type CryptoConfig struct {
	// Version is the version of the config.
	Version string `yaml:"version,omitempty" env:"VERSION,overwrite"`

	// Crypto variables
	KeyTTL         time.Duration `yaml:"key_ttl,omitempty" env:"KEY_TTL,overwrite"`
	GracePeriod    time.Duration `yaml:"grace_period,omitempty" env:"GRACE_PERIOD,overwrite"`
	DisabledPeriod time.Duration `yaml:"disabled_period,omitempty" env:"DISABLED_PERIOD,overwrite"`
}

// Validate checks if the config is valid.
func (cfg *CryptoConfig) Validate() error {
	cfg.SetDefault()

	var err error
	if cfg.Version != Version {
		err = multierror.Append(err, fmt.Errorf("unexpected Version %q want %q", cfg.Version, Version))
	}

	if cfg.KeyTTL == 0 {
		err = multierror.Append(err, fmt.Errorf("key ttl is 0"))
	}

	if cfg.GracePeriod == 0 {
		err = multierror.Append(err, fmt.Errorf("grace period is 0"))
	}

	if cfg.DisabledPeriod == 0 {
		err = multierror.Append(err, fmt.Errorf("disabled period is 0"))
	}

	return err
}

// SetDefault sets default for the config.
func (cfg *CryptoConfig) SetDefault() {
	// TODO: set defaults for other fields if necessary.
	if cfg.Version == "" {
		cfg.Version = Version
	}
}

// GetRotationAge gets the duration after a key has been created that a new key should be created.
func (cfg *CryptoConfig) GetRotationAge() time.Duration {
	return cfg.KeyTTL - cfg.GracePeriod
}

// GetDestroyAge gets the duration after a key has been created when it becomes a candidate to be destroyed.
func (cfg *CryptoConfig) GetDestroyAge() time.Duration {
	return cfg.KeyTTL + cfg.DisabledPeriod
}

// LoadConfig reads in a yaml file, applies ENV config overrides, and finally validates the config.
func LoadConfig(ctx context.Context, b []byte) (*CryptoConfig, error) {
	cfg := &CryptoConfig{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, err
	}

	// Process overrides from env vars.
	l := envconfig.PrefixLookuper("JVS_", envconfig.OsLookuper())
	if err := envconfig.ProcessWith(ctx, cfg, l); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed validating config: %w", err)
	}

	return cfg, nil
}
