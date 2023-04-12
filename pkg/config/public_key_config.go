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

package config

import (
	"fmt"
	"time"

	"github.com/abcxyz/pkg/cli"
	"github.com/hashicorp/go-multierror"
)

// PublicKeyConfig is the config used for public key hosting.
type PublicKeyConfig struct {
	// ProjectID is the Google Cloud project ID.
	ProjectID string `env:"PROJECT_ID"`

	// DevMode controls enables more granular debugging in logs.
	DevMode bool `env:"DEV_MODE,default=false"`

	// TODO: This is intended to be temporary, and will eventually be retrieved from a persistent external datastore
	// https://github.com/abcxyz/jvs/issues/17
	// KeyName format: `projects/*/locations/*/keyRings/*/cryptoKeys/*`
	// https://pkg.go.dev/google.golang.org/genproto/googleapis/cloud/kms/v1#PublicKeyKey
	KeyNames     []string      `yaml:"key_names,omitempty" env:"KEY_NAMES,overwrite"`
	Port         string        `env:"PORT,default=8080"`
	CacheTimeout time.Duration `yaml:"cache_timeout" env:"CACHE_TIMEOUT, default=5m"`
}

func (cfg *PublicKeyConfig) Validate() error {
	var merr *multierror.Error

	if cfg.ProjectID == "" {
		merr = multierror.Append(merr, fmt.Errorf("empty ProjectID"))
	}

	if len(cfg.KeyNames) == 0 {
		merr = multierror.Append(merr, fmt.Errorf("empty KeyNames"))
	}

	if got := cfg.CacheTimeout; got <= 0 {
		merr = multierror.Append(merr, fmt.Errorf("cache_timeout must be a positive duration, got %q", got))
	}

	return merr.ErrorOrNil()
}

// ToFlags binds the config to the give [cli.FlagSet] and returns it.
func (cfg *PublicKeyConfig) ToFlags(set *cli.FlagSet) *cli.FlagSet {
	// Command options
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

	f = set.NewSection("KEY OPTIONS")

	f.StringSliceVar(&cli.StringSliceVar{
		Name:    "key-names",
		Target:  &cfg.KeyNames,
		EnvVar:  "KEY_NAMES",
		Example: "projects/[JVS_PROJECT]/locations/global/keyRings/[JVS_KEYRING]/cryptoKeys/[JVS_KEY]",
		Usage:   "List of KMS key names",
	})

	f.DurationVar(&cli.DurationVar{
		Name:    "cache-timeout",
		Target:  &cfg.CacheTimeout,
		EnvVar:  "CACHE_TIMEOUT",
		Default: 5 * time.Minute,
		Usage:   "The duration that a KMS key will be cached.",
	})

	return set
}
