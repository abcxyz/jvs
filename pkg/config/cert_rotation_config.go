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
	"fmt"
	"time"

	"github.com/abcxyz/pkg/cli"
	"github.com/hashicorp/go-multierror"
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
	KeyTTL time.Duration `yaml:"key_ttl,omitempty" env:"KEY_TTL,overwrite"`

	// GracePeriod is a length of time between when we rotate the key and when an old Key Version is no longer valid and available
	GracePeriod time.Duration `yaml:"grace_period,omitempty" env:"GRACE_PERIOD,overwrite"`

	// PropagationDelay is the time that it takes for a change in the key in KMS to be reflected in the client.
	PropagationDelay time.Duration `yaml:"propagation_delay,omitempty" env:"PROPAGATION_DELAY,overwrite"`

	// DisabledPeriod is a time between when the key is disabled, and when we delete the key.
	DisabledPeriod time.Duration `yaml:"disabled_period,omitempty" env:"DISABLED_PERIOD,overwrite"`

	// TODO: This is intended to be temporary, and will eventually be retrieved from a persistent external datastore
	// https://github.com/abcxyz/jvs/issues/17
	// KeyName format: `projects/*/locations/*/keyRings/*/cryptoKeys/*`
	// https://pkg.go.dev/google.golang.org/genproto/googleapis/cloud/kms/v1#CryptoKey
	KeyNames []string `yaml:"key_names,omitempty" env:"KEY_NAMES,overwrite"`
}

// Validate checks if the config is valid.
func (cfg *CertRotationConfig) Validate() error {
	var err *multierror.Error

	if cfg.KeyTTL <= 0 {
		err = multierror.Append(err, fmt.Errorf("key ttl must be a positive duration, got %q", cfg.KeyTTL))
	}

	if cfg.GracePeriod <= 0 {
		err = multierror.Append(err, fmt.Errorf("grace period must be a positive duration, got %q", cfg.GracePeriod))
	}

	if cfg.DisabledPeriod <= 0 {
		err = multierror.Append(err, fmt.Errorf("disabled period must be a positive duration, got %q", cfg.DisabledPeriod))
	}

	// Propagation delay must be positive but less than the grace period.
	if cfg.PropagationDelay <= 0 {
		err = multierror.Append(err, fmt.Errorf("propagation delay must be a positive duration, got %q", cfg.PropagationDelay))
	}
	if cfg.PropagationDelay > cfg.GracePeriod {
		err = multierror.Append(err, fmt.Errorf("propagation delay %q must be less than grace period %q",
			cfg.PropagationDelay, cfg.GracePeriod))
	}

	return err.ErrorOrNil()
}

// RotationAge gets the duration after a key has been created that a new key should be created.
func (cfg *CertRotationConfig) RotationAge() time.Duration {
	return cfg.KeyTTL - cfg.GracePeriod
}

// DestroyAge gets the duration after a key has been created when it becomes a candidate to be destroyed.
func (cfg *CertRotationConfig) DestroyAge() time.Duration {
	return cfg.KeyTTL + cfg.DisabledPeriod
}

// ToFlags returns a [cli.FlagSet] that is bound to the config.
func (cfg *CertRotationConfig) ToFlags(opts ...cli.Option) *cli.FlagSet {
	set := cli.NewFlagSet(opts...)

	// Command options
	f := set.NewSection("COMMON SERVER OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:   "project-id",
		Target: &cfg.ProjectID,
		EnvVar: "PROJECT_ID",
		Usage:  `Google Cloud project ID.`,
	})

	f.BoolVar(&cli.BoolVar{
		Name:    "dev-mode",
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
		EnvVar:  "KEY_TTL",
		Default: 10 * time.Minute,
		Usage:   "The time that a key will be valid for.",
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "grace-period",
		Target:  &cfg.GracePeriod,
		EnvVar:  "GRACE_PERIOD",
		Default: 5 * time.Minute,
		Usage:   "The time between when we rotate the key and when the old key version is no longer available.",
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "propagation-delay",
		Target:  &cfg.PropagationDelay,
		EnvVar:  "PROPAGATION_DELAY",
		Default: 5 * time.Minute,
		Usage:   "The time that it takes for a key change to be reflected in the client.",
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "disable-period",
		Target:  &cfg.DisabledPeriod,
		EnvVar:  "DISABLED_PERIOD",
		Default: 2 * time.Minute,
		Usage:   "The time between when the key is disabled and when we delete the key.",
	})

	f.StringSliceVar(&cli.StringSliceVar{
		Name:    "key-names",
		Target:  &cfg.KeyNames,
		EnvVar:  "KEY_NAMES",
		Example: "projects/[JVS_PROJECT]/locations/global/keyRings/[JVS_KEYRING]/cryptoKeys/[JVS_KEY]",
		Usage:   "List of KMS key names",
	})

	return set
}
