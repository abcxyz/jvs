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

package v1alpha1

import "time"

const (
	// Version of the API and config.
	Version = "v1alpha1"
)

// CryptoConfig is the full jvs config.
type CryptoConfig struct {
	// Version is the version of the config.
	Version string `yaml:"version,omitempty" env:"VERSION,overwrite"`

	// Crypto variables
	KeyTTL          time.Duration `yaml:"key_ttl_days,omitempty"`
	PropagationTime time.Duration `yaml:"propagation_time_minutes,omitempty"`
	GracePeriod     time.Duration `yaml:"grace_period_minutes,omitempty"`
	DisabledPeriod  time.Duration `yaml:"disabled_period_days,omitempty"`
}

// Validate checks if the config is valid.
func (cfg *CryptoConfig) Validate() error {
	// TODO https://github.com/abcxyz/jvs/issues/2
	return nil
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
