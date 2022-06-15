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

	"github.com/hashicorp/go-multierror"
)

const (
	// DefaultCLIConfigVersion is the default CLI config version.
	DefaultCLIConfigVersion = 1
)

type CLIConfig struct {
	// Version is the version of the config.
	Version uint8 `yaml:"version,omitempty" env:"VERSION,overwrite"`

	// Server is the JVS server address.
	Server string `yaml:"server,omitempty" env:"SERVER,overwrite"`
}

// Validate checks if the config is valid.
func (cfg *CLIConfig) Validate() error {
	cfg.SetDefault()

	var err *multierror.Error
	if cfg.Server == "" {
		err = multierror.Append(err, fmt.Errorf("missing JVS server address"))
	}

	return err.ErrorOrNil()
}

// SetDefault sets default for the config.
func (cfg *CLIConfig) SetDefault() {
	if cfg.Version == 0 {
		cfg.Version = DefaultCLIConfigVersion
	}
}
