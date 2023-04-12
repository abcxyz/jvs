// Copyright 2023 The Authors (see AUTHORS file)
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

	"github.com/abcxyz/pkg/cli"
	"github.com/hashicorp/go-multierror"
)

// UIServiceConfig defines the set over environment variables required
// for running this application.
type UIServiceConfig struct {
	*JustificationConfig

	Allowlist []string `env:"JVS_UI_ALLOWLIST,required"`
}

// Validate checks if the config is valid.
func (cfg *UIServiceConfig) Validate() error {
	var merr *multierror.Error

	if err := cfg.JustificationConfig.Validate(); err != nil {
		merr = multierror.Append(merr, err)
	}

	if len(cfg.Allowlist) == 0 {
		merr = multierror.Append(merr, fmt.Errorf("empty Allowlist"))
	}

	// edge case, exclusive asterisk(*)
	if !(len(cfg.Allowlist) == 1 && cfg.Allowlist[0] == "*") {
		// confirm no asterisks if muiltiple values provided
		// i.e. ["example.com" "*"] is invalid
		for _, e := range cfg.Allowlist {
			if e == "*" {
				merr = multierror.Append(merr,
					fmt.Errorf("asterisk(*) must be exclusive, no other domains allowed"))
			}
		}
	}

	return merr.ErrorOrNil()
}

// ToFlags binds the config to the give [cli.FlagSet] and returns it.
func (cfg *UIServiceConfig) ToFlags(set *cli.FlagSet) *cli.FlagSet {
	if cfg.JustificationConfig == nil {
		cfg.JustificationConfig = &JustificationConfig{}
	}
	set = cfg.JustificationConfig.ToFlags(set)

	f := set.NewSection("UI OPTIONS")
	f.StringSliceVar(&cli.StringSliceVar{
		Name:    "allowlist",
		Target:  &cfg.Allowlist,
		EnvVar:  "JVS_UI_ALLOWLIST",
		Example: "example.com,*.foo.bar",
		Usage:   "List of allowed domains.",
	})

	return set
}
