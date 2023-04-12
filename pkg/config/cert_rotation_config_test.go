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

func TestCertRotationConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name       string
		cfg        string
		envs       map[string]string
		wantConfig *CertRotationConfig
		wantErr    string
	}{
		{
			name: "all_values_specified",
			cfg: `key_ttl: 720h # 30 days
grace_period: 2h
disabled_period: 720h # 30 days
propagation_delay: 1h
`,
			wantConfig: &CertRotationConfig{
				Port:             "8080",
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
			wantConfig: &CertRotationConfig{
				Port:             "8080",
				KeyTTL:           720 * time.Hour, // 30 days
				GracePeriod:      2 * time.Hour,   // 2 hours
				DisabledPeriod:   720 * time.Hour, // 30 days
				PropagationDelay: time.Hour,       // 1 hour
			},
		},
		{
			name:       "test_empty_key_ttl",
			cfg:        ``,
			wantConfig: nil,
			wantErr:    `key ttl must be a positive duration, got "0s"`,
		},
		{
			name:       "test_empty_grace_period",
			cfg:        ``,
			wantConfig: nil,
			wantErr:    `grace period must be a positive duration, got "0s"`,
		},
		{
			name:       "test_empty_disabled_period",
			cfg:        ``,
			wantConfig: nil,
			wantErr:    `disabled period must be a positive duration, got "0s"`,
		},
		{
			name:       "test_empty_propagation_delay",
			cfg:        ``,
			wantConfig: nil,
			wantErr:    `propagation delay must be a positive duration, got "0s"`,
		},

		{
			name: "test_negative_key_ttl",
			cfg: `
key_ttl: -720h
`,
			wantConfig: nil,
			wantErr:    `key ttl must be a positive duration, got "-720h0m0s"`,
		},
		{
			name: "test_negative_grace_period",
			cfg: `
grace_period: -2h
`,
			wantConfig: nil,
			wantErr:    `grace period must be a positive duration, got "-2h0m0s"`,
		},
		{
			name: "test_negative_disabled_period",
			cfg: `
disabled_period: -720h
`,
			wantConfig: nil,
			wantErr:    `disabled period must be a positive duration, got "-720h0m0s"`,
		},
		{
			name: "test_negative_propagation_delay",
			cfg: `
propagation_delay: -1h
`,
			wantConfig: nil,
			wantErr:    `propagation delay must be a positive duration, got "-1h0m0s"`,
		},
		{
			name: "test_propagation_delay_greater_than_grace_period",
			cfg: `
grace_period: 1h
propagation_delay: 2h
`,
			wantConfig: nil,
			wantErr:    `propagation delay "2h0m0s" must be less than grace period "1h0m0s"`,
		},
		{
			name: "all_values_specified_env_override",
			cfg: `key_ttl: 720h # 30 days
grace_period: 2h
disabled_period: 720h # 30 days
propagation_delay: 1h
`,
			envs: map[string]string{
				"KEY_TTL":      "1080h", // 45 days
				"GRACE_PERIOD": "4h",
			},
			wantConfig: &CertRotationConfig{
				Port:             "8080",
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
				"KEY_TTL":           "1080h", // 45 days
				"GRACE_PERIOD":      "4h",
				"DISABLED_PERIOD":   "1080h", // 45 days
				"PROPAGATION_DELAY": "1h",
			},
			wantConfig: &CertRotationConfig{
				Port:             "8080",
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
			gotConfig := &CertRotationConfig{}
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

func TestCertRotationConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("validates_crypto_config", func(t *testing.T) {
		t.Parallel()

		cfg := &CertRotationConfig{
			GracePeriod: -1 * time.Minute, // must be positive, should fail validation
		}

		err := cfg.Validate()
		if diff := testutil.DiffErrString(err, "grace period must be a positive duration"); diff != "" {
			t.Errorf(diff)
		}
	})
}

func TestRotationAge(t *testing.T) {
	t.Parallel()

	cfg := &CertRotationConfig{
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

	cfg := &CertRotationConfig{
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
