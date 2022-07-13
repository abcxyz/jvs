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
	CurrentVersion            = 1
	SignerCacheTimeoutDefault = 5 * time.Minute
	IssuerDefault             = "jvs.abcxyz.dev"
)

// JustificationConfig is the full jvs config.
type JustificationConfig struct {
	// Version is the version of the config.
	Version uint8 `yaml:"version,omitempty" env:"VERSION,overwrite"`

	// Service configuration.
	Port string `yaml:"port,omitempty" env:"PORT,overwrite"`

	// ProjectID is the ID of GCP project where the Firestore documents with the KMS key locates
	ProjectID string `yaml:"project_id,omitempty" env:"PROJECT_ID,overwrite"`

	// SignerCacheTimeout is the duration that keys stay in cache before being revoked.
	SignerCacheTimeout time.Duration `yaml:"signer_cache_timeout" env:"SIGNER_CACHE_TIMEOUT,overwrite"`

	// Issuer will be used to set the issuer field when signing JWTs
	Issuer string `yaml:"issuer" env:"ISSUER,overwrite"`
}

// Validate checks if the config is valid.
func (cfg *JustificationConfig) Validate() error {
	cfg.SetDefault()
	var err *multierror.Error
	if cfg.Version != CurrentVersion {
		err = multierror.Append(err, fmt.Errorf("unexpected Version %d want %d", cfg.Version, CurrentVersion))
	}
	if cfg.SignerCacheTimeout <= 0 {
		err = multierror.Append(err, fmt.Errorf("cache timeout invalid: %d", cfg.SignerCacheTimeout))
	}
	if cfg.ProjectID == "" {
		err = multierror.Append(err, fmt.Errorf("blank project id is invalid"))
	}
	return err.ErrorOrNil()
}

// SetDefault sets default for the config.
func (cfg *JustificationConfig) SetDefault() {
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.Version == 0 {
		cfg.Version = Version
	}
	if cfg.SignerCacheTimeout == 0 {
		// env config lib doesn't gracefully handle env overrides with defaults, have to set manually.
		cfg.SignerCacheTimeout = SignerCacheTimeoutDefault
	}
	if cfg.Issuer == "" {
		cfg.Issuer = IssuerDefault
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
