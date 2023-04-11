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

	"github.com/abcxyz/pkg/timeutil"
	"github.com/hashicorp/go-multierror"
)

// JustificationConfigVersions is the list of allowed versions for the
// JustificationConfig.
var JustificationConfigVersions = NewVersionList("1")

// JustificationConfig is the full jvs config.
type JustificationConfig struct {
	// ProjectID is the Google Cloud project ID.
	ProjectID string `env:"PROJECT_ID"`

	// Version is the version of the config.
	Version string `yaml:"version,omitempty" env:"VERSION,overwrite,default=1"`

	// Service configuration.
	Port string `yaml:"port,omitempty" env:"PORT,overwrite,default=8080"`

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
	var err *multierror.Error

	if !JustificationConfigVersions.Contains(cfg.Version) {
		err = multierror.Append(err, fmt.Errorf("version %q is invalid, valid versions are: %q",
			cfg.Version, JustificationConfigVersions.List()))
	}

	if cfg.SignerCacheTimeout <= 0 {
		err = multierror.Append(err, fmt.Errorf("cache timeout must be a positive duration, got %s",
			cfg.SignerCacheTimeout))
	}

	if def, max := cfg.DefaultTTL, cfg.MaxTTL; def > max {
		err = multierror.Append(err, fmt.Errorf("default ttl (%s) must be less than or equal to the max ttl (%s)",
			timeutil.HumanDuration(def), timeutil.HumanDuration(max)))
	}

	return err.ErrorOrNil()
}
