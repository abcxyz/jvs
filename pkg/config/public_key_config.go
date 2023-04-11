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

func (c *PublicKeyConfig) Validate() error {
	var err *multierror.Error

	if got := c.CacheTimeout; got <= 0 {
		err = multierror.Append(err, fmt.Errorf("cache_timeout must be a positive duration, got %q", got))
	}

	return err.ErrorOrNil()
}
