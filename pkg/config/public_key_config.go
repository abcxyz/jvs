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
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v2"
)

// PublicKeyConfig is the config used for public key hosting.
type PublicKeyConfig struct {
	// ProjectID is the ID of GCP project where the Firestore documents with the KMS key locates
	ProjectID    string        `yaml:"project_ID,omitempty" env:"PROJECT_ID,overwrite"`
	CacheTimeout time.Duration `yaml:"cache_timeout" env:"CACHE_TIMEOUT"`
	Port         string        `env:"PORT,default=8080"`
}

// LoadConfig calls the necessary methods to load in config using the OsLookuper which finds env variables specified on the host.
func LoadPublicKeyConfig(ctx context.Context, b []byte) (*PublicKeyConfig, error) {
	return loadPublicKeyConfigFromLookuper(ctx, b, envconfig.OsLookuper())
}

// loadConfigFromLooker reads in a yaml file, applies ENV config overrides from the lookuper, and finally validates the config.
func loadPublicKeyConfigFromLookuper(ctx context.Context, b []byte, lookuper envconfig.Lookuper) (*PublicKeyConfig, error) {
	cfg := &PublicKeyConfig{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	// Process overrides from env vars.
	l := envconfig.PrefixLookuper("", lookuper)
	if err := envconfig.ProcessWith(ctx, cfg, l); err != nil {
		return nil, fmt.Errorf("failed to process environment: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed validating config: %w", err)
	}
	return cfg, nil
}

// Validate checks if the config is valid.
func (cfg *PublicKeyConfig) Validate() error {
	var err *multierror.Error
	if cfg.ProjectID == "" {
		err = multierror.Append(err, fmt.Errorf("blank project id is invalid"))
	}
	return err.ErrorOrNil()
}
