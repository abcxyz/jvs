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
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/abcxyz/pkg/cfgloader"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-envconfig"
)

func TestJustificationConfig_Defaults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var justificationConfig JustificationConfig
	if err := envconfig.ProcessWith(ctx, &justificationConfig, envconfig.MapLookuper(nil)); err != nil {
		t.Fatal(err)
	}

	want := &JustificationConfig{
		Port:               "8080",
		SignerCacheTimeout: 5 * time.Minute,
		Issuer:             "jvs.abcxyz.dev",
		DefaultTTL:         15 * time.Minute,
		MaxTTL:             4 * time.Hour,
	}

	if diff := cmp.Diff(want, &justificationConfig); diff != "" {
		t.Errorf("config with defaults (-want, +got):\n%s", diff)
	}
}

func TestLoadJustificationConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name       string
		cfg        string
		envs       map[string]string
		wantConfig *JustificationConfig
		wantErr    string
	}{
		{
			name: "all_values_specified",
			cfg: `
port: 123
signer_cache_timeout: 1m
issuer: jvs
default_ttl: 5m
max_ttl: 30m
`,
			wantConfig: &JustificationConfig{
				Port:               "123",
				SignerCacheTimeout: 1 * time.Minute,
				Issuer:             "jvs",
				DefaultTTL:         5 * time.Minute,
				MaxTTL:             30 * time.Minute,
			},
		},
		{
			name: "test_default",
			cfg:  ``,
			wantConfig: &JustificationConfig{
				Port:               "8080",
				SignerCacheTimeout: 5 * time.Minute,
				Issuer:             "jvs.abcxyz.dev",
				DefaultTTL:         15 * time.Minute,
				MaxTTL:             4 * time.Hour,
				DevMode:            false,
			},
		},
		{
			name: "invalid_signer_cache_timeout",
			cfg: `
signer_cache_timeout: -1m
`,
			wantConfig: nil,
			wantErr:    `cache timeout must be a positive duration, got -1m0s`,
		},
		{
			name: "default_ttl_greater_than_max_ttl",
			cfg: `
default_ttl: 1h
max_ttl: 30m
`,
			wantConfig: nil,
			wantErr:    `default ttl (1h) must be less than or equal to the max ttl (30m)`,
		},
		{
			name: "all_values_specified_env_override",
			cfg: `
port: 8080
signer_cache_timeout: 1m
issuer: jvs
default_ttl: 15m
max_ttl: 1h
`,
			envs: map[string]string{
				"PORT":                 "tcp",
				"SIGNER_CACHE_TIMEOUT": "2m",
				"ISSUER":               "other",
				"DEFAULT_TTL":          "30m",
				"MAX_TTL":              "2h",
				"DEV_MODE":             "true",
			},
			wantConfig: &JustificationConfig{
				Port:               "tcp",
				SignerCacheTimeout: 2 * time.Minute,
				Issuer:             "other",
				DefaultTTL:         30 * time.Minute,
				MaxTTL:             2 * time.Hour,
				DevMode:            true,
			},
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lookuper := envconfig.MapLookuper(tc.envs)
			content := bytes.NewBufferString(tc.cfg).Bytes()
			gotConfig := &JustificationConfig{}
			err := cfgloader.Load(ctx, gotConfig,
				cfgloader.WithLookuper(lookuper), cfgloader.WithYAML(content))
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(tc.wantConfig, gotConfig); diff != "" {
				t.Errorf("Config unexpected diff (-want,+got):\n%s", diff)
			}
		})
	}
}
