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

	"github.com/google/go-cmp/cmp"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/testutil"
)

func TestPublicKeyConfig_ToFlags(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		envs       map[string]string
		wantConfig *PublicKeyConfig
	}{
		{
			name: "all_values_specified",
			envs: map[string]string{
				"PROJECT_ID":                   "example-project",
				"DEV_MODE":                     "true",
				"PORT":                         "0",
				"JVS_KEY_NAMES":                "fake/key",
				"JVS_PUBLIC_KEY_CACHE_TIMEOUT": "10m",
			},
			wantConfig: &PublicKeyConfig{
				ProjectID:    "example-project",
				DevMode:      true,
				Port:         "0",
				KeyNames:     []string{"fake/key"},
				CacheTimeout: 10 * time.Minute,
			},
		},
		{
			name: "default_values",
			wantConfig: &PublicKeyConfig{
				Port:         "8080",
				CacheTimeout: 5 * time.Minute,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotConfig := &PublicKeyConfig{}
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

func TestPublicKeyConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     *PublicKeyConfig
		wantErr string
	}{
		{
			name: "valid",
			cfg: &PublicKeyConfig{
				ProjectID:    "example-project",
				Port:         "8080",
				KeyNames:     []string{"fake/key"},
				CacheTimeout: 5 * time.Minute,
			},
		},
		{
			name: "empty_project",
			cfg: &PublicKeyConfig{
				Port:         "8080",
				KeyNames:     []string{"fake/key"},
				CacheTimeout: 5 * time.Minute,
			},
			wantErr: "empty ProjectID",
		},
		{
			name: "empty_key_names",
			cfg: &PublicKeyConfig{
				ProjectID:    "example-project",
				Port:         "8080",
				CacheTimeout: 5 * time.Minute,
			},
			wantErr: "empty KeyNames",
		},
		{
			name: "invalid_cache_timeout",
			cfg: &PublicKeyConfig{
				ProjectID:    "example-project",
				Port:         "8080",
				KeyNames:     []string{"fake/key"},
				CacheTimeout: -5 * time.Minute,
			},
			wantErr: "cache_timeout must be a positive duration",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
		})
	}
}
