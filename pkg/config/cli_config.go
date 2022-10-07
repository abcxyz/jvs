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
	"strings"

	"github.com/hashicorp/go-multierror"
)

// CLIConfigVersions is the list of allowed versions for the CLIConfig.
var CLIConfigVersions = NewVersionList("1")

type CLIConfig struct {
	// Version is the version of the config.
	Version string `yaml:"version,omitempty"`

	// Server is the JVS server address.
	Server string `yaml:"server,omitempty"`

	// Insecure indicates whether the CLI should allow an insecure connection to
	// the server.
	Insecure bool `yaml:"insecure,omitempty"`

	// JWKSEndpoint is the full path (including protocol and port) to the JWKS
	// endpoint on a JVS server (e.g. https://example.com/.well-known/jwks).
	JWKSEndpoint string `yaml:"jwks_endpoint,omitempty" mapstructure:"jwks_endpoint"`
}

// Validate checks if the config is valid.
func (cfg *CLIConfig) Validate() error {
	cfg.SetDefault()

	var err *multierror.Error

	if !CLIConfigVersions.Contains(cfg.Version) {
		err = multierror.Append(err, fmt.Errorf("version %q is invalid, valid versions are: %q",
			cfg.Version, CLIConfigVersions.List()))
	}

	return err.ErrorOrNil()
}

// SetDefault sets default for the config.
func (cfg *CLIConfig) SetDefault() {
	if cfg.Version == "" {
		cfg.Version = "1"
	}

	// Default to server's doman if JWKSEndpoint is not specified.
	if cfg.JWKSEndpoint == "" && cfg.Server != "" {
		cfg.JWKSEndpoint = fmt.Sprintf("https://%s:8080/.well-known/jwks", strings.Split(cfg.Server, ":")[0])
	}
}
