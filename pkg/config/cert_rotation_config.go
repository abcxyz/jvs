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

import "github.com/hashicorp/go-multierror"

// CertRotationConfig is a configuration for cert rotation services.
type CertRotationConfig struct {
	*CryptoConfig

	// ProjectID is the Google Cloud project ID.
	ProjectID string `env:"PROJECT_ID"`

	// DevMode controls enables more granular debugging in logs.
	DevMode bool `env:"DEV_MODE,default=false"`

	// Port is the port where the service runs.
	Port string `env:"PORT,default=8080"`
}

// Validate checks if the config is valid.
func (c *CertRotationConfig) Validate() error {
	var merr *multierror.Error

	if err := c.CryptoConfig.Validate(); err != nil {
		merr = multierror.Append(merr, err)
	}

	return merr.ErrorOrNil()
}
