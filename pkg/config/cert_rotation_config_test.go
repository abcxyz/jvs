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

func TestCertRotationConfig_ToFlags(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		envs       map[string]string
		wantConfig *CertRotationConfig
	}{
		{
			name: "all_values_specified",
			envs: map[string]string{
				"PROJECT_ID":                     "example-project",
				"DEV_MODE":                       "true",
				"PORT":                           "0",
				"JVS_ROTATION_KEY_TTL":           "15m",
				"JVS_ROTATION_GRACE_PERIOD":      "10m",
				"JVS_ROTATION_PROPAGATION_DELAY": "10m",
				"JVS_ROTATION_DISABLED_PERIOD":   "3m",
				"JVS_KEY_NAMES":                  "fake/key",
			},
			wantConfig: &CertRotationConfig{
				ProjectID:        "example-project",
				DevMode:          true,
				Port:             "0",
				KeyTTL:           15 * time.Minute,
				GracePeriod:      10 * time.Minute,
				PropagationDelay: 10 * time.Minute,
				DisabledPeriod:   3 * time.Minute,
				KeyNames:         []string{"fake/key"},
			},
		},
		{
			name: "default_values",
			wantConfig: &CertRotationConfig{
				Port:             "8080",
				KeyTTL:           10 * time.Minute,
				GracePeriod:      5 * time.Minute,
				PropagationDelay: 5 * time.Minute,
				DisabledPeriod:   2 * time.Minute,
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotConfig := &CertRotationConfig{}
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

func TestCertRotationConfig_Validate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cfg     *CertRotationConfig
		wantErr string
	}{
		{
			name: "valid",
			cfg: &CertRotationConfig{
				ProjectID:        "example-project",
				Port:             "8080",
				KeyTTL:           10 * time.Minute,
				GracePeriod:      5 * time.Minute,
				PropagationDelay: 5 * time.Minute,
				DisabledPeriod:   2 * time.Minute,
				KeyNames:         []string{"fake/key"},
			},
		},
		{
			name: "empty_project",
			cfg: &CertRotationConfig{
				Port:             "8080",
				KeyTTL:           10 * time.Minute,
				GracePeriod:      5 * time.Minute,
				PropagationDelay: 5 * time.Minute,
				DisabledPeriod:   2 * time.Minute,
				KeyNames:         []string{"fake/key"},
			},
			wantErr: "empty ProjectID",
		},
		{
			name: "empty_key_names",
			cfg: &CertRotationConfig{
				ProjectID:        "example-project",
				Port:             "8080",
				KeyTTL:           10 * time.Minute,
				GracePeriod:      5 * time.Minute,
				PropagationDelay: 5 * time.Minute,
				DisabledPeriod:   2 * time.Minute,
			},
			wantErr: "empty KeyNames",
		},
		{
			name: "invalid_key_ttl",
			cfg: &CertRotationConfig{
				ProjectID:        "example-project",
				Port:             "8080",
				KeyTTL:           -10 * time.Minute,
				GracePeriod:      5 * time.Minute,
				PropagationDelay: 5 * time.Minute,
				DisabledPeriod:   2 * time.Minute,
				KeyNames:         []string{"fake/key"},
			},
			wantErr: "key ttl must be a positive duration",
		},
		{
			name: "invalid_grace_period",
			cfg: &CertRotationConfig{
				ProjectID:        "example-project",
				Port:             "8080",
				KeyTTL:           10 * time.Minute,
				GracePeriod:      -5 * time.Minute,
				PropagationDelay: 5 * time.Minute,
				DisabledPeriod:   2 * time.Minute,
				KeyNames:         []string{"fake/key"},
			},
			wantErr: "grace period must be a positive duration",
		},
		{
			name: "invalid_disable_period",
			cfg: &CertRotationConfig{
				ProjectID:        "example-project",
				Port:             "8080",
				KeyTTL:           10 * time.Minute,
				GracePeriod:      5 * time.Minute,
				PropagationDelay: 5 * time.Minute,
				DisabledPeriod:   -2 * time.Minute,
				KeyNames:         []string{"fake/key"},
			},
			wantErr: "disabled period must be a positive duration",
		},
		{
			name: "invalid_propagation_delay",
			cfg: &CertRotationConfig{
				ProjectID:        "example-project",
				Port:             "8080",
				KeyTTL:           10 * time.Minute,
				GracePeriod:      5 * time.Minute,
				PropagationDelay: -5 * time.Minute,
				DisabledPeriod:   2 * time.Minute,
				KeyNames:         []string{"fake/key"},
			},
			wantErr: "propagation delay must be a positive duration",
		},
		{
			name: "invalid_propagation_delay_greater_than_grace_period",
			cfg: &CertRotationConfig{
				ProjectID:        "example-project",
				Port:             "8080",
				KeyTTL:           10 * time.Minute,
				GracePeriod:      1 * time.Minute,
				PropagationDelay: 5 * time.Minute,
				DisabledPeriod:   2 * time.Minute,
				KeyNames:         []string{"fake/key"},
			},
			wantErr: "must be less than grace period",
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
