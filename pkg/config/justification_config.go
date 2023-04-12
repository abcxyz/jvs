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
	"fmt"
	"time"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/timeutil"
	"github.com/hashicorp/go-multierror"
)

// JustificationConfig is the full jvs config.
type JustificationConfig struct {
	// ProjectID is the Google Cloud project ID.
	ProjectID string `env:"PROJECT_ID"`

	// Service configuration.
	Port string `yaml:"port,omitempty" env:"PORT,overwrite,default=8080"`

	// DevMode enables more granular debugging in logs.
	DevMode bool `env:"DEV_MODE,default=false"`

	// KeyName format: `projects/*/locations/*/keyRings/*/cryptoKeys/*`
	// https://pkg.go.dev/google.golang.org/genproto/googleapis/cloud/kms/v1#CryptoKey
	KeyName string `yaml:"key,omitempty" env:"KEY,overwrite"`

	// SignerCacheTimeout is the duration that keys stay in cache before being revoked.
	SignerCacheTimeout time.Duration `yaml:"signer_cache_timeout" env:"SIGNER_CACHE_TIMEOUT,overwrite,default=5m"`

	// Issuer will be used to set the issuer field when signing JWTs
	Issuer string `yaml:"issuer" env:"ISSUER,overwrite,default=jvs.abcxyz.dev"`

	// DefaultTTL sets the default TTL for JVS tokens that do not explicitly
	// request a TTL. MaxTTL is the system-configured maximum TTL that a token can
	// request.
	//
	// The DefaultTTL must be less than or equal to MaxTTL.
	DefaultTTL time.Duration `yaml:"default_ttl" env:"DEFAULT_TTL,overwrite,default=15m"`
	MaxTTL     time.Duration `yaml:"max_ttl" env:"MAX_TTL,overwrite,default=4h"`
}

// Validate checks if the config is valid.
func (cfg *JustificationConfig) Validate() error {
	var merr *multierror.Error

	if cfg.ProjectID == "" {
		merr = multierror.Append(merr, fmt.Errorf("empty ProjectID"))
	}

	if cfg.KeyName == "" {
		merr = multierror.Append(merr, fmt.Errorf("empty KeyName"))
	}

	if got := cfg.SignerCacheTimeout; got <= 0 {
		merr = multierror.Append(merr, fmt.Errorf("cache timeout must be a positive duration, got %s",
			got))
	}

	if def, max := cfg.DefaultTTL, cfg.MaxTTL; def > max {
		merr = multierror.Append(merr, fmt.Errorf("default ttl (%s) must be less than or equal to the max ttl (%s)",
			timeutil.HumanDuration(def), timeutil.HumanDuration(max)))
	}

	return merr.ErrorOrNil()
}

// ToFlags binds the config to the give [cli.FlagSet] and returns it.
func (cfg *JustificationConfig) ToFlags(set *cli.FlagSet) *cli.FlagSet {
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

	f = set.NewSection("API OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "key-name",
		Target:  &cfg.KeyName,
		EnvVar:  "KEY",
		Example: "projects/[JVS_PROJECT]/locations/global/keyRings/[JVS_KEYRING]/cryptoKeys/[JVS_KEY]",
		Usage:   `The KMS key for signing JVS tokens.`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "issuer",
		Target:  &cfg.Issuer,
		EnvVar:  "ISSUER",
		Default: "jvs.abcxyz.dev",
		Usage:   `The value to set to the issuer claim when signing JVS tokens.`,
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "signer-cache-timeout",
		Target:  &cfg.SignerCacheTimeout,
		EnvVar:  "SIGNER_CACHE_TIMEOUT",
		Default: 5 * time.Minute,
		Usage:   "The duration that keys stay in cache before being revoked.",
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "default-ttl",
		Target:  &cfg.DefaultTTL,
		EnvVar:  "DEFAULT_TTL",
		Default: 15 * time.Minute,
		Usage:   "The default TTL for JVS tokens if not specified in the request.",
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "max-ttl",
		Target:  &cfg.MaxTTL,
		EnvVar:  "MAX_TTL",
		Default: 4 * time.Hour,
		Usage:   "The maximum TTL that a token can have.",
	})

	return set
}
