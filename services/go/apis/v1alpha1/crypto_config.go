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

const (
	// Version of the API and config.
	Version = "v1alpha1"
)

// CryptoConfig is the full jvs config.
type CryptoConfig struct {
	// Version is the version of the config.
	Version string `yaml:"version,omitempty" env:"VERSION,overwrite"`

	// Crypto variables
	KeyTTLDays             uint64 `yaml:"key_ttl_days,omitempty"`
	PropagationTimeMinutes uint64 `yaml:"propagation_time_minutes,omitempty"`
	GracePeriodMinutes     uint64 `yaml:"grace_period_minutes,omitempty"`
	DisabledPeriodDays     uint64 `yaml:"disabled_period_days,omitempty"`
}

// Validate checks if the config is valid.
func (cfg *CryptoConfig) Validate() error {
	// TODO
	return nil
}

// SetDefault sets default for the config.
func (cfg *CryptoConfig) SetDefault() {
	// TODO: set defaults for other fields if necessary.
	if cfg.Version == "" {
		cfg.Version = Version
	}
}

func (cfg *CryptoConfig) GetRotationAgeSeconds() uint64 {
	ttlSeconds := cfg.KeyTTLDays * 24 * 60 * 60
	graceSeconds := cfg.GracePeriodMinutes * 60
	return ttlSeconds - graceSeconds
}

func (cfg *CryptoConfig) GetDisableAgeSeconds() uint64 {
	return cfg.KeyTTLDays * 24 * 60 * 60
}

func (cfg *CryptoConfig) GetDestroyAgeSeconds() uint64 {
	ttlSeconds := cfg.KeyTTLDays * 24 * 60 * 60
	disabledPeriod := cfg.DisabledPeriodDays * 24 * 60 * 60
	return ttlSeconds + disabledPeriod
}
