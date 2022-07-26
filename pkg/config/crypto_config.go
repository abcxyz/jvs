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

// CryptoConfigVersions is the list of allowed versions for the CryptoConfig.
var CryptoConfigVersions = NewVersionList("1")

// CryptoConfig is the full jvs config.
type CryptoConfig struct {
	// Version is the version of the config.
	Version string `yaml:"version,omitempty" env:"VERSION,overwrite,default=1"`

	// -- Crypto variables --
	// KeyTTL is the length of time that we expect a key to be valid for.
	KeyTTL time.Duration `yaml:"key_ttl,omitempty" env:"KEY_TTL,overwrite"`
	// GracePeriod is a length of time between when we rotate the key and when an old Key Version is no longer valid and available
	GracePeriod time.Duration `yaml:"grace_period,omitempty" env:"GRACE_PERIOD,overwrite"`
	// PropagationDelay is the time that it takes for a change in the key in KMS to be reflected in the client.
	PropagationDelay time.Duration `yaml:"propagation_delay,omitempty" env:"PROPAGATION_DELAY,overwrite"`
	// DisabledPeriod is a time between when the key is disabled, and when we delete the key.
	DisabledPeriod time.Duration `yaml:"disabled_period,omitempty" env:"DISABLED_PERIOD,overwrite"`

	// FirestoreDocName is the resource name of the Firestore Document which stores the KMS key names
	// Format: projects/{project_id}/databases/{databaseId}/documents/{document_path}
	// Example: "projects/test-project/databases/(default)/documents/jvs/key_config"
	FirestoreDocResourceName string `yaml:"firestore_doc_resource_name,omitempty" env:"FIRESTORE_DOC_RESOURCE_NAME,overwrite"`
}

// Validate checks if the config is valid.
func (cfg *CryptoConfig) Validate() error {
	var err *multierror.Error

	if !CryptoConfigVersions.Contains(cfg.Version) {
		err = multierror.Append(err, fmt.Errorf("version %q is invalid, valid versions are: %q",
			cfg.Version, CryptoConfigVersions.List()))
	}

	if cfg.KeyTTL <= 0 {
		err = multierror.Append(err, fmt.Errorf("key ttl must be a positive duration, got %q", cfg.KeyTTL))
	}

	if cfg.GracePeriod <= 0 {
		err = multierror.Append(err, fmt.Errorf("grace period must be a positive duration, got %q", cfg.GracePeriod))
	}

	if cfg.DisabledPeriod <= 0 {
		err = multierror.Append(err, fmt.Errorf("disabled period must be a positive duration, got %q", cfg.DisabledPeriod))
	}

	// Propagation delay must be positive but less than than grace period.
	if cfg.PropagationDelay <= 0 {
		err = multierror.Append(err, fmt.Errorf("propagation delay must be a positive duration, got %q", cfg.PropagationDelay))
	}
	if cfg.PropagationDelay > cfg.GracePeriod {
		err = multierror.Append(err, fmt.Errorf("propagation delay %q must be less than grace period %q",
			cfg.PropagationDelay, cfg.GracePeriod))
	}

	if cfg.FirestoreDocResourceName == "" {
		err = multierror.Append(err, fmt.Errorf("firestore doc resource name must be non-empty"))
	}

	return err.ErrorOrNil()
}

// GetRotationAge gets the duration after a key has been created that a new key should be created.
func (cfg *CryptoConfig) RotationAge() time.Duration {
	return cfg.KeyTTL - cfg.GracePeriod
}

// GetDestroyAge gets the duration after a key has been created when it becomes a candidate to be destroyed.
func (cfg *CryptoConfig) DestroyAge() time.Duration {
	return cfg.KeyTTL + cfg.DisabledPeriod
}

// LoadConfig calls the necessary methods to load in config using the OsLookuper which finds env variables specified on the host.
func LoadCryptoConfig(ctx context.Context, b []byte) (*CryptoConfig, error) {
	return loadCryptoConfigFromLookuper(ctx, b, envconfig.OsLookuper())
}

// loadConfigFromLooker reads in a yaml file, applies ENV config overrides from the lookuper, and finally validates the config.
func loadCryptoConfigFromLookuper(ctx context.Context, b []byte, lookuper envconfig.Lookuper) (*CryptoConfig, error) {
	cfg := &CryptoConfig{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// Process overrides from env vars.
	l := envconfig.PrefixLookuper("JVS_", lookuper)
	if err := envconfig.ProcessWith(ctx, cfg, l); err != nil {
		return nil, fmt.Errorf("failed to process environment: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed validating config: %w", err)
	}

	return cfg, nil
}
