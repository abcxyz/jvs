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

	"github.com/abcxyz/jvs/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name       string
		cfg        string
		envs       map[string]string
		wantConfig *CryptoConfig
		wantErr    string
	}{
		{
			name: "all_values_specified",
			cfg: `
version: v1alpha1
key_ttl: 720h # 30 days
grace_period: 2h
disabled_period: 720h # 30 days
`,
			wantConfig: &CryptoConfig{
				Version:        "v1alpha1",
				KeyTTL:         30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
				GracePeriod:    2 * 60 * 60 * 1_000_000_000,       // 2 hours
				DisabledPeriod: 30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
			},
		},
		{
			name: "test_default",
			cfg: `
key_ttl: 720h # 30 days
propagation_time: 30m
grace_period: 2h
disabled_period: 720h # 30 days
`,
			wantConfig: &CryptoConfig{
				Version:        "v1alpha1",
				KeyTTL:         30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
				GracePeriod:    2 * 60 * 60 * 1_000_000_000,       // 2 hours
				DisabledPeriod: 30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
			},
		},
		{
			name: "test_wrong_version",
			cfg: `
version: wrong_ver
key_ttl: 720h # 30 days
grace_period: 2h
disabled_period: 720h # 30 days
`,
			wantConfig: &CryptoConfig{
				Version:        "wrong_ver",
				KeyTTL:         30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
				GracePeriod:    2 * 60 * 60 * 1_000_000_000,       // 2 hours
				DisabledPeriod: 30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
			},
			wantErr: "failed validating config: 1 error occurred:\n\t* unexpected Version \"wrong_ver\" want \"v1alpha1\"\n\n",
		},
		{
			name: "test_empty_ttl",
			cfg: `
version: v1alpha1
grace_period: 2h
disabled_period: 720h # 30 days
`,
			wantConfig: &CryptoConfig{
				Version:        "v1alpha1",
				KeyTTL:         0,
				GracePeriod:    2 * 60 * 60 * 1_000_000_000,       // 2 hours
				DisabledPeriod: 30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
			},
			wantErr: "failed validating config: 1 error occurred:\n\t* key ttl is 0\n\n",
		},
		{
			name: "test_empty",
			cfg:  "",
			wantConfig: &CryptoConfig{
				Version:        "v1alpha1",
				KeyTTL:         0,
				GracePeriod:    0,
				DisabledPeriod: 0,
			},
			wantErr: "failed validating config: 3 errors occurred:\n\t* key ttl is 0\n\t* grace period is 0\n\t* disabled period is 0\n\n",
		},
		{
			name: "all_values_specified_env_override",
			cfg: `
version: v1alpha1
key_ttl: 720h # 30 days
grace_period: 2h
disabled_period: 720h # 30 days
`,
			envs: map[string]string{
				"JVS_KEY_TTL":      "1080h", // 45 days
				"JVS_GRACE_PERIOD": "4h",
			},
			wantConfig: &CryptoConfig{
				Version:        "v1alpha1",
				KeyTTL:         45 * 24 * 60 * 60 * 1_000_000_000, // 45 days
				GracePeriod:    4 * 60 * 60 * 1_000_000_000,       // 4 hours
				DisabledPeriod: 30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
			},
		},
		{
			name: "non_default_values_specified_in_envs",
			cfg:  ``,
			envs: map[string]string{
				"JVS_KEY_TTL":         "1080h", // 45 days
				"JVS_GRACE_PERIOD":    "4h",
				"JVS_DISABLED_PERIOD": "1080h", // 45 days
			},
			wantConfig: &CryptoConfig{
				Version:        "v1alpha1",
				KeyTTL:         45 * 24 * 60 * 60 * 1_000_000_000, // 45 days
				GracePeriod:    4 * 60 * 60 * 1_000_000_000,       // 4 hours
				DisabledPeriod: 45 * 24 * 60 * 60 * 1_000_000_000, // 45 days
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// No parallel due to testing with env vars.
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}
			content := bytes.NewBufferString(tc.cfg).Bytes()
			gotConfig, err := LoadConfig(ctx, content)
			if err != nil {
				testutil.ErrCmp(t, tc.wantErr, err)
			} else if diff := cmp.Diff(tc.wantConfig, gotConfig); diff != "" {
				t.Errorf("Config unexpected diff (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestGetRotationAge(t *testing.T) {
	t.Parallel()
	cfg := &CryptoConfig{
		Version:        "v1alpha1",
		KeyTTL:         30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
		GracePeriod:    2 * 60 * 60 * 1_000_000_000,       // 2 hours
		DisabledPeriod: 30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
	}
	expected, err := time.ParseDuration("718h") // 29 days, 22 hours
	if err != nil {
		t.Error("Couldn't parse duration")
	}
	if diff := cmp.Diff(cfg.GetRotationAge(), expected); diff != "" {
		t.Errorf("unexpected rotation age (-want,+got):\n%s", diff)
	}
}

func TestGetDestroyAge(t *testing.T) {
	t.Parallel()
	cfg := &CryptoConfig{
		Version:        "v1alpha1",
		KeyTTL:         30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
		GracePeriod:    2 * 60 * 60 * 1_000_000_000,       // 2 hours
		DisabledPeriod: 30 * 24 * 60 * 60 * 1_000_000_000, // 30 days
	}
	expected, err := time.ParseDuration("1440h") // 60 days
	if err != nil {
		t.Error("Couldn't parse duration")
	}
	if diff := cmp.Diff(cfg.GetDestroyAge(), expected); diff != "" {
		t.Errorf("unexpected destroy age (-want,+got):\n%s", diff)
	}
}
