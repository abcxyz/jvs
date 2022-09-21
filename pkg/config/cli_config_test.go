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

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestCLIConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *CLIConfig
		wantErr string
	}{
		{
			name: "no_error",
			cfg: &CLIConfig{
				Server:       "example.com",
				JWKSEndpoint: "https://jvs.corp:8080/.well-known/jwks",
			},
		},
		{
			name: "bad_version",
			cfg: &CLIConfig{
				Version: "255",
			},
			wantErr: "missing JVS server address",
		},
		{
			name: "missing_server_error",
			cfg: &CLIConfig{
				JWKSEndpoint: "https://jvs.corp:8080/.well-known/jwks",
			},
			wantErr: "missing JVS server address",
		},
		{
			name: "missing_jwks_endpoint",
			cfg: &CLIConfig{
				Server: "example.com",
			},
			wantErr: "missing JWKS endpoint",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.cfg.Validate()
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected err: %s", diff)
			}
		})
	}
}

func TestCLIConfig_SetDefault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *CLIConfig
		wantCfg *CLIConfig
	}{{
		name: "default_empty_authentication",
		cfg:  &CLIConfig{},
		wantCfg: &CLIConfig{
			Version:        "1",
			Authentication: &CLIAuthentication{},
		},
	}}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.cfg.SetDefault()
			if diff := cmp.Diff(tc.wantCfg, tc.cfg); diff != "" {
				t.Errorf("config with defaults (-want,+got):\n%s", diff)
			}
		})
	}
}
