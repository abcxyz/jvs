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
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/abcxyz/pkg/cli"
)

// UIServiceConfig defines the set over environment variables required
// for running this application.
type UIServiceConfig struct {
	*JustificationConfig

	Allowlist []string `env:"JVS_UI_ALLOWLIST,required"`
}

// Validate checks if the config is valid.
func (cfg *UIServiceConfig) Validate() (merr error) {
	if err := cfg.JustificationConfig.Validate(); err != nil {
		merr = errors.Join(merr, err)
	}

	if len(cfg.Allowlist) == 0 {
		merr = errors.Join(merr, fmt.Errorf("empty Allowlist"))
	}

	// edge case, exclusive asterisk(*)
	if !(len(cfg.Allowlist) == 1 && strings.TrimSpace(cfg.Allowlist[0]) == "*") {
		// confirm no asterisks if multiple values provided
		// i.e. ["example.com", "*"] is invalid
		for _, e := range cfg.Allowlist {
			trimmed := strings.TrimSpace(e)
			if trimmed == "*" {
				merr = errors.Join(merr,
					fmt.Errorf("asterisk(*) must be exclusive, no other domains allowed"))
				continue
			}
			if err := ValidateAllowlistEntry(trimmed); err != nil {
				merr = errors.Join(merr, fmt.Errorf("invalid allowlist entry %q: %w", e, err))
			}
		}
	}

	return merr
}

// ValidateAllowlistEntry validates a single non-wildcard-all entry in JVS_UI_ALLOWLIST.
// Entries may be an IP literal, a hostname (e.g. "example.com" or "localhost"),
// or a wildcard domain pattern (e.g. "*.example.com").
func ValidateAllowlistEntry(rawEntry string) error {
	entry := strings.ToLower(strings.TrimSpace(rawEntry))
	if entry == "" {
		return fmt.Errorf("entry is empty")
	}
	if net.ParseIP(entry) != nil {
		return nil
	}
	if strings.ContainsAny(entry, ":/?#@\\ \t\r\n") {
		return fmt.Errorf("must be a bare domain, wildcard domain (*.example.com), or IP literal without scheme, port, or path")
	}
	if strings.HasPrefix(entry, "*.") {
		base := strings.TrimPrefix(entry, "*.")
		if !strings.Contains(base, ".") || !IsValidHostname(base) {
			return fmt.Errorf("wildcard entry must have a valid multi-label base domain")
		}
		return nil
	}
	if !IsValidHostname(entry) {
		return fmt.Errorf("invalid hostname")
	}
	return nil
}

// IsValidHostname reports whether host is a valid lowercase ASCII RFC-1123 DNS hostname
// without leading/trailing dots, empty labels, wildcards, or an all-numeric TLD.
func IsValidHostname(host string) bool {
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	if strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			ch := label[i]
			if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
				return false
			}
		}
	}
	// Reject multi-label hosts whose final label is all digits (e.g. malformed IPv4).
	if len(labels) > 1 {
		last := labels[len(labels)-1]
		allDigits := true
		for i := 0; i < len(last); i++ {
			if last[i] < '0' || last[i] > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return false
		}
	}
	return true
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
