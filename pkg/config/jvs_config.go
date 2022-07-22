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

// JustificationConfigVersions is the list of allowed versions for the
// JustificationConfig.
var JustificationConfigVersions = NewVersionList("1")

// JustificationConfig is the full jvs config.
type JustificationConfig struct {
	// Version is the version of the config.
	Version string `yaml:"version,omitempty" env:"VERSION,overwrite,default=1"`

	// Service configuration.
	Port string `yaml:"port,omitempty" env:"PORT,overwrite,default=8080"`

	FirestoreProjectID string `yaml:"firestore_project_id,omitempty" env:"FIRESTORE_PROJECT_ID,overwrite"`

	// SignerCacheTimeout is the duration that keys stay in cache before being revoked.
	SignerCacheTimeout time.Duration `yaml:"signer_cache_timeout" env:"SIGNER_CACHE_TIMEOUT,overwrite,default=5m"`

	// Issuer will be used to set the issuer field when signing JWTs
	Issuer string `yaml:"issuer" env:"ISSUER,overwrite,default=jvs.abcxyz.dev"`
}

// Validate checks if the config is valid.
func (cfg *JustificationConfig) Validate() error {
	var err *multierror.Error

	if !JustificationConfigVersions.Contains(cfg.Version) {
		err = multierror.Append(err, fmt.Errorf("version %q is invalid, valid versions are: %q",
			cfg.Version, JustificationConfigVersions.List()))
	}

	if cfg.SignerCacheTimeout <= 0 {
		err = multierror.Append(err, fmt.Errorf("cache timeout must be a positive duration, got %s",
			cfg.SignerCacheTimeout))
	}

	if cfg.FirestoreProjectID == "" {
		err = multierror.Append(err, fmt.Errorf("firestore project id can't be empty"))
	}
	return err.ErrorOrNil()
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

type JVSKeyConfig struct {
	// KeyName format: `[projects/*/locations/*/keyRings/*/cryptoKeys/*]`
	KeyName string `yaml:"key_name,omitempty" firestore:"key_name,omitempty"`
}
