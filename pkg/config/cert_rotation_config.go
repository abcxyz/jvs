// Copyright 2023 Google LLC
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
	"errors"
	"fmt"
	"time"

	"github.com/abcxyz/pkg/cli"
)

// CertRotationConfig is a configuration for cert rotation services.
type CertRotationConfig struct {
	// ProjectID is the Google Cloud project ID.
	ProjectID string `env:"PROJECT_ID"`

	// DevMode controls enables more granular debugging in logs.
	DevMode bool `env:"DEV_MODE,default=false"`

	// Port is the port where the service runs.
	Port string `env:"PORT,default=8080"`

	// -- Crypto variables --
	// KeyTTL is the length of time that we expect a key to be valid for.
	KeyTTL time.Duration `env:"JVS_ROTATION_KEY_TTL,overwrite"`

	// GracePeriod is a length of time between when we rotate the key and when an old Key Version is no longer valid and available
	GracePeriod time.Duration `env:"JVS_ROTATION_GRACE_PERIOD,overwrite"`

	// PropagationDelay is the time that it takes for a change in the key in KMS to be reflected in the client.
	PropagationDelay time.Duration `env:"JVS_ROTATION_PROPAGATION_DELAY,overwrite"`

	// DisabledPeriod is a time between when the key is disabled, and when we delete the key.
	DisabledPeriod time.Duration `env:"JVS_ROTATION_DISABLED_PERIOD,overwrite"`

	// KeyName format: `projects/*/locations/*/keyRings/*/cryptoKeys/*`
	// https://pkg.go.dev/google.golang.org/genproto/googleapis/cloud/kms/v1#CryptoKey
	KeyNames []string `env:"JVS_KEY_NAMES,overwrite"`
}

// Validate checks if the config is valid.
func (cfg *CertRotationConfig) Validate() (merr error) {
	if cfg.ProjectID == "" {
		merr = errors.Join(merr, fmt.Errorf("empty ProjectID"))
	}

	if len(cfg.KeyNames) == 0 {
		merr = errors.Join(merr, fmt.Errorf("empty KeyNames"))
	}

	if got := cfg.KeyTTL; got <= 0 {
		merr = errors.Join(merr, fmt.Errorf("key ttl must be a positive duration, got %q", got))
	}

	if got := cfg.GracePeriod; got <= 0 {
		merr = errors.Join(merr, fmt.Errorf("grace period must be a positive duration, got %q", got))
	}

	if got := cfg.DisabledPeriod; got <= 0 {
		merr = errors.Join(merr, fmt.Errorf("disabled period must be a positive duration, got %q", got))
	}

	// Propagation delay must be positive but less than the grace period.
	if got := cfg.PropagationDelay; got <= 0 {
		merr = errors.Join(merr, fmt.Errorf("propagation delay must be a positive duration, got %q", got))
	}

	if cfg.PropagationDelay > cfg.GracePeriod {
		merr = errors.Join(merr, fmt.Errorf("propagation delay %q must be less than grace period %q",
			cfg.PropagationDelay, cfg.GracePeriod))
	}

	return
}

// RotationAge gets the duration after a key has been created that a new key should be created.
func (cfg *CertRotationConfig) RotationAge() time.Duration {
	return cfg.KeyTTL - cfg.GracePeriod
}

// DestroyAge gets the duration after a key has been created when it becomes a candidate to be destroyed.
func (cfg *CertRotationConfig) DestroyAge() time.Duration {
	return cfg.KeyTTL + cfg.DisabledPeriod
}

// ToFlags binds the config to the give [cli.FlagSet] and returns it.
func (cfg *CertRotationConfig) ToFlags(set *cli.FlagSet) *cli.FlagSet {
	f := set.NewSection("COMMON SERVER OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:   "project-id",
		Target: &cfg.ProjectID,
		EnvVar: "PROJECT_ID",
		Usage:  `Google Cloud project ID.`,
	})

	f.BoolVar(&cli.BoolVar{
		Name:    "dev",
		Target:  &cfg.DevMode,
		EnvVar:  "DEV_MODE",
		Default: false,
		Usage:   "Set to true to enable more granular debugging in logs",
	})

	f.StringVar(&cli.StringVar{
		Name:    "port",
		Target:  &cfg.Port,
		EnvVar:  "PORT",
		Default: "8080",
		Usage:   `The port the server listens to.`,
	})

	f = set.NewSection("KEY ROTATION OPTIONS")

	f.DurationVar(&cli.DurationVar{
		Name:    "key-ttl",
		Target:  &cfg.KeyTTL,
		EnvVar:  "JVS_ROTATION_KEY_TTL",
		Default: 10 * time.Minute,
		Usage:   "The time that a key will be valid for.",
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "grace-period",
		Target:  &cfg.GracePeriod,
		EnvVar:  "JVS_ROTATION_GRACE_PERIOD",
		Default: 5 * time.Minute,
		Usage:   "The time between when we rotate the key and when the old key version is no longer available.",
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "propagation-delay",
		Target:  &cfg.PropagationDelay,
		EnvVar:  "JVS_ROTATION_PROPAGATION_DELAY",
		Default: 5 * time.Minute,
		Usage:   "The time that it takes for a key change to be reflected in the client.",
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "disable-period",
		Target:  &cfg.DisabledPeriod,
		EnvVar:  "JVS_ROTATION_DISABLED_PERIOD",
		Default: 2 * time.Minute,
		Usage:   "The time between when the key is disabled and when we delete the key.",
	})

	f.StringSliceVar(&cli.StringSliceVar{
		Name:    "key-names",
		Target:  &cfg.KeyNames,
		EnvVar:  "JVS_KEY_NAMES",
		Example: "projects/[JVS_PROJECT]/locations/global/keyRings/[JVS_KEYRING]/cryptoKeys/[JVS_KEY]",
		Usage:   "List of KMS key names",
	})

	return set
}
