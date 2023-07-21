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
	"testing"
	"time"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestJustificationConfig_ToFlags(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		envs       map[string]string
		wantConfig *JustificationConfig
	}{
		{
			name: "all_values_specified",
			envs: map[string]string{
				"PROJECT_ID":                   "example-project",
				"DEV_MODE":                     "true",
				"PORT":                         "0",
				"JVS_KEY":                      "fake/key",
				"JVS_API_SIGNER_CACHE_TIMEOUT": "10m",
				"JVS_API_ISSUER":               "example.com",
				"JVS_PLUGIN_DIR":               "/var/jvs/pluginsDir",
				"JVS_API_DEFAULT_TTL":          "30m",
				"JVS_API_MAX_TTL":              "8h",
			},
			wantConfig: &JustificationConfig{
				ProjectID:          "example-project",
				DevMode:            true,
				Port:               "0",
				KeyName:            "fake/key",
				SignerCacheTimeout: 10 * time.Minute,
				Issuer:             "example.com",
				PluginDir:          "/var/jvs/pluginsDir",
				DefaultTTL:         30 * time.Minute,
				MaxTTL:             8 * time.Hour,
			},
		},
		{
			name: "default_values",
			wantConfig: &JustificationConfig{
				Port:               "8080",
				SignerCacheTimeout: 5 * time.Minute,
				Issuer:             "jvs.abcxyz.dev",
				PluginDir:          "/var/jvs/plugins",
				DefaultTTL:         15 * time.Minute,
				MaxTTL:             4 * time.Hour,
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotConfig := &JustificationConfig{}
			set := cli.NewFlagSet(cli.WithLookupEnv(cli.MapLookuper(tc.envs)))
			set = gotConfig.ToFlags(set)
			if err := set.Parse([]string{}); err != nil {
				t.Errorf("unexpected flag set parse error: %v", err)
			}
			if diff := cmp.Diff(tc.wantConfig, gotConfig); diff != "" {
				t.Errorf("Config unexpected diff (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestJustificationConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     *JustificationConfig
		wantErr string
	}{
		{
			name: "valid",
			cfg: &JustificationConfig{
				ProjectID:          "example-project",
				Port:               "8080",
				KeyName:            "fake/key",
				SignerCacheTimeout: 5 * time.Minute,
				Issuer:             "jvs.abcxyz.dev",
				PluginDir:          "/var/jvs/pluginsDir",
				DefaultTTL:         15 * time.Minute,
				MaxTTL:             4 * time.Hour,
			},
		},
		{
			name: "relative_plugin_dir",
			cfg: &JustificationConfig{
				ProjectID:          "example-project",
				Port:               "8080",
				KeyName:            "fake/key",
				SignerCacheTimeout: 5 * time.Minute,
				Issuer:             "jvs.abcxyz.dev",
				PluginDir:          "./pluginsDir",
				DefaultTTL:         15 * time.Minute,
				MaxTTL:             4 * time.Hour,
			},
		},
		{
			name: "empty_project",
			cfg: &JustificationConfig{
				Port:               "8080",
				KeyName:            "fake/key",
				SignerCacheTimeout: 5 * time.Minute,
				Issuer:             "jvs.abcxyz.dev",
				PluginDir:          "/var/jvs/pluginsDir",
				DefaultTTL:         15 * time.Minute,
				MaxTTL:             4 * time.Hour,
			},
			wantErr: "empty ProjectID",
		},
		{
			name: "empty_key_name",
			cfg: &JustificationConfig{
				ProjectID:          "example-project",
				Port:               "8080",
				SignerCacheTimeout: 5 * time.Minute,
				Issuer:             "jvs.abcxyz.dev",
				PluginDir:          "/var/jvs/pluginsDir",
				DefaultTTL:         15 * time.Minute,
				MaxTTL:             4 * time.Hour,
			},
			wantErr: "empty KeyName",
		},
		{
			name: "invalid_cache_timeout",
			cfg: &JustificationConfig{
				ProjectID:          "example-project",
				Port:               "8080",
				KeyName:            "fake/key",
				SignerCacheTimeout: -5 * time.Minute,
				Issuer:             "jvs.abcxyz.dev",
				PluginDir:          "/var/jvs/pluginsDir",
				DefaultTTL:         15 * time.Minute,
				MaxTTL:             4 * time.Hour,
			},
			wantErr: "cache timeout must be a positive duration",
		},
		{
			name: "invalid_default_ttl",
			cfg: &JustificationConfig{
				ProjectID:          "example-project",
				Port:               "8080",
				KeyName:            "fake/key",
				SignerCacheTimeout: 5 * time.Minute,
				Issuer:             "jvs.abcxyz.dev",
				PluginDir:          "/var/jvs/pluginsDir",
				DefaultTTL:         15 * time.Minute,
				MaxTTL:             10 * time.Minute,
			},
			wantErr: "must be less than or equal to the max ttl",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
		})
	}
}
