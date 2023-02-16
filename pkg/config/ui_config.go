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
	"context"
	"fmt"
	"regexp"

	"github.com/abcxyz/pkg/cfgloader"
	"github.com/sethvargo/go-envconfig"
)

// UIServiceConfig defines the set over environment variables required
// for running this application.
type UIServiceConfig struct {
	Port      string   `env:"PORT,default=9091"`
	AllowList []string `env:"ALLOW_LIST,delimiter=;,required"`
	DevMode   bool     `env:"DEV_MODE,default=false"`
}

var validRegexPattern = regexp.MustCompile(`^(([\w-]+\.)|(\*\.))+[\w-]+$`)

// NewUIConfig creates a new UIServiceConfig from environment variables.
func NewUIConfig(ctx context.Context) (*UIServiceConfig, error) {
	return newUIConfig(ctx, envconfig.OsLookuper())
}

func newUIConfig(ctx context.Context, lu envconfig.Lookuper) (*UIServiceConfig, error) {
	var cfg UIServiceConfig
	if err := cfgloader.Load(ctx, &cfg, cfgloader.WithLookuper(lu)); err != nil {
		return nil, fmt.Errorf("failed to parse server config: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the config is valid.
func (cfg *UIServiceConfig) Validate() error {
	// edge case, exclusive asterisk(*)
	if len(cfg.AllowList) == 1 && cfg.AllowList[0] == "*" {
		return nil
	}

	// confirm no asterisks if muiltiple values provided
	// i.e. ["example.com" "*"] is invalid
	for _, e := range cfg.AllowList {
		if e == "*" {
			return fmt.Errorf("asterisk(*) must be exclusive, no other domains allowed")
		}
	}

	// validate the AllowList entries are valid
	for _, domain := range cfg.AllowList {
		if !validRegexPattern.MatchString(domain) {
			return fmt.Errorf("domain in the ALLOW_LIST is invalid: %s", domain)
		}
	}

	return nil
}
