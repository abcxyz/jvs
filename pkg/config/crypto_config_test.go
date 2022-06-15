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

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-envconfig"
)

func TestLoadCryptoConfig(t *testing.T) {
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
version: 1
key_ttl: 720h # 30 days
grace_period: 2h
disabled_period: 720h # 30 days
propagation_delay: 1h
`,
			wantConfig: &CryptoConfig{
				Version:          1,
				KeyTTL:           720 * time.Hour, // 30 days
				GracePeriod:      2 * time.Hour,   // 2 hours
				DisabledPeriod:   720 * time.Hour, // 30 days
				PropagationDelay: time.Hour,       // 1 hour
			},
		},
		{
			name: "test_default",
			cfg: `
key_ttl: 720h # 30 days
propagation_time: 30m
grace_period: 2h
disabled_period: 720h # 30 days
propagation_delay: 1h
`,
			wantConfig: &CryptoConfig{
				Version:          1,
				KeyTTL:           720 * time.Hour, // 30 days
				GracePeriod:      2 * time.Hour,   // 2 hours
				DisabledPeriod:   720 * time.Hour, // 30 days
				PropagationDelay: time.Hour,       // 1 hour
			},
		},
		{
			name: "test_wrong_version",
			cfg: `
version: 255
key_ttl: 720h # 30 days
grace_period: 2h
disabled_period: 720h # 30 days
propagation_delay: 1h
`,
			wantConfig: nil,
			wantErr:    "failed validating config: 1 error occurred:\n\t* unexpected Version 255 want 1\n\n",
		},
		{
			name: "test_invalid_propagation_delay",
			cfg: `
version: 1
key_ttl: 720h # 30 days
grace_period: 2h
disabled_period: 720h # 30 days
propagation_delay: 3h
`,
			wantConfig: nil,
			wantErr:    "failed validating config: 1 error occurred:\n\t* propagation delay is invalid: 3h0m0s\n\n",
		},
		{
			name: "test_empty_ttl",
			cfg: `
version: 1
grace_period: 2h
disabled_period: 720h # 30 days
propagation_delay: 1h
`,
			wantConfig: nil,
			wantErr:    "failed validating config: 1 error occurred:\n\t* key ttl is invalid: 0s\n\n",
		},
		{
			name:       "test_empty",
			cfg:        "",
			wantConfig: nil,
			wantErr:    "failed validating config: 4 errors occurred:\n\t* key ttl is invalid: 0s\n\t* grace period is invalid: 0s\n\t* disabled period is invalid: 0s\n\t* propagation delay is invalid: 0s\n\n",
		},
		{
			name: "test_negative",
			cfg: `
version: 1
key_ttl: -720h
grace_period: -2h
disabled_period: -720h
propagation_delay: -1h
`,
			wantConfig: nil,
			wantErr:    "failed validating config: 4 errors occurred:\n\t* key ttl is invalid: -720h0m0s\n\t* grace period is invalid: -2h0m0s\n\t* disabled period is invalid: -720h0m0s\n\t* propagation delay is invalid: -1h0m0s\n\n",
		},
		{
			name: "all_values_specified_env_override",
			cfg: `
version: 1
key_ttl: 720h # 30 days
grace_period: 2h
disabled_period: 720h # 30 days
propagation_delay: 1h
`,
			envs: map[string]string{
				"JVS_KEY_TTL":      "1080h", // 45 days
				"JVS_GRACE_PERIOD": "4h",
			},
			wantConfig: &CryptoConfig{
				Version:          1,
				KeyTTL:           1080 * time.Hour, // 45 days
				GracePeriod:      4 * time.Hour,    // 4 hours
				DisabledPeriod:   720 * time.Hour,  // 30 days
				PropagationDelay: time.Hour,        // 1 hour
			},
		},
		{
			name: "non_default_values_specified_in_envs",
			cfg:  ``,
			envs: map[string]string{
				"JVS_KEY_TTL":           "1080h", // 45 days
				"JVS_GRACE_PERIOD":      "4h",
				"JVS_DISABLED_PERIOD":   "1080h", // 45 days
				"JVS_PROPAGATION_DELAY": "1h",
			},
			wantConfig: &CryptoConfig{
				Version:          1,
				KeyTTL:           1080 * time.Hour, // 45 days
				GracePeriod:      4 * time.Hour,    // 4 hours
				DisabledPeriod:   1080 * time.Hour, // 45 days
				PropagationDelay: time.Hour,        // 1 hour
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			lookuper := envconfig.MapLookuper(tc.envs)
			content := bytes.NewBufferString(tc.cfg).Bytes()
			gotConfig, err := loadCryptoConfigFromLookuper(ctx, content, lookuper)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if diff := cmp.Diff(tc.wantConfig, gotConfig); diff != "" {
				t.Errorf("Config unexpected diff (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestRotationAge(t *testing.T) {
	t.Parallel()
	cfg := &CryptoConfig{
		Version:        1,
		KeyTTL:         720 * time.Hour, // 30 days
		GracePeriod:    2 * time.Hour,   // 2 hours
		DisabledPeriod: 720 * time.Hour, // 30 days
	}
	expected, err := time.ParseDuration("718h") // 29 days, 22 hours
	if err != nil {
		t.Error("Couldn't parse duration")
	}
	if expected != cfg.RotationAge() {
		t.Errorf("unexpected rotation age. Want: %s, but got: %s\n", expected, cfg.RotationAge())
	}
}

func TestDestroyAge(t *testing.T) {
	t.Parallel()
	cfg := &CryptoConfig{
		Version:        1,
		KeyTTL:         720 * time.Hour, // 30 days
		GracePeriod:    2 * time.Hour,   // 2 hours
		DisabledPeriod: 720 * time.Hour, // 30 days
	}
	expected, err := time.ParseDuration("1440h") // 60 days
	if err != nil {
		t.Error("Couldn't parse duration")
	}
	if expected != cfg.DestroyAge() {
		t.Errorf("unexpected destroy age. Want: %s, but got: %s\n", expected, cfg.DestroyAge())
	}
}
