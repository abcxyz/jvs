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

package client

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-envconfig"
)

func TestLoadJVSConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name       string
		cfg        string
		envs       map[string]string
		wantConfig *JVSConfig
		wantErr    string
	}{
		{
			name: "all_values_specified",
			cfg: `
version: 1
endpoint: https://jvs.corp:8080/.well-known/jwks
cache_timeout: 1m
allow_breakglass: true
`,
			wantConfig: &JVSConfig{
				Version:         "1",
				JWKSEndpoint:    "https://jvs.corp:8080/.well-known/jwks",
				CacheTimeout:    time.Minute,
				AllowBreakglass: true,
			},
		},
		{
			name: "test_default",
			cfg: `
endpoint: https://jvs.corp:8080/.well-known/jwks
`,
			wantConfig: &JVSConfig{
				Version:         "1",
				JWKSEndpoint:    "https://jvs.corp:8080/.well-known/jwks",
				CacheTimeout:    5 * time.Minute,
				AllowBreakglass: false,
			},
		},
		{
			name: "test_wrong_version",
			cfg: `
version: 255
endpoint: https://jvs.corp:8080/.well-known/jwks
cache_timeout: 1m
allow_breakglass: true
`,
			wantConfig: nil,
			wantErr:    `version "255" is invalid, valid versions are:`,
		},
		{
			name: "test_invalid_timeout",
			cfg: `
version: 1
endpoint: https://jvs.corp:8080/.well-known/jwks
cache_timeout: -1m
allow_breakglass: true
`,
			wantConfig: nil,
			wantErr:    `cache timeout must be a positive duration, got "-1m0s"`,
		},
		{
			name: "all_values_specified_env_override",
			cfg: `
version: 1
endpoint: https://jvs.corp:8080/.well-known/jwks
cache_timeout: 1m
allow_breakglass: false
`,
			envs: map[string]string{
				"JVS_VERSION":          "1",
				"JVS_ENDPOINT":         "other.net:443",
				"JVS_CACHE_TIMEOUT":    "2m",
				"JVS_ALLOW_BREAKGLASS": "true",
			},
			wantConfig: &JVSConfig{
				Version:         "1",
				JWKSEndpoint:    "other.net:443",
				CacheTimeout:    2 * time.Minute,
				AllowBreakglass: true,
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			lookuper := envconfig.MapLookuper(tc.envs)
			content := bytes.NewBufferString(tc.cfg).Bytes()
			gotConfig, err := loadJVSConfigFromLookuper(ctx, content, lookuper)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if diff := cmp.Diff(tc.wantConfig, gotConfig); diff != "" {
				t.Errorf("Config unexpected diff (-want,+got):\n%s", diff)
			}
		})
	}
}
