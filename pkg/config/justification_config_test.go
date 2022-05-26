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
	"github.com/sethvargo/go-envconfig"
)

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
version: 1
signer_cache_timeout: 1m
issuer: jvs
`,
			wantConfig: &JustificationConfig{
				Port:               "123",
				Version:            1,
				SignerCacheTimeout: 1 * time.Minute,
				Issuer:             "jvs",
			},
		},
		{
			name: "test_default",
			cfg:  ``,
			wantConfig: &JustificationConfig{
				Port:               "8080",
				Version:            1,
				SignerCacheTimeout: 5 * time.Minute,
				Issuer:             "jvs.abcxyz.dev",
			},
		},
		{
			name: "test_wrong_version",
			cfg: `
version: 255
`,
			wantConfig: nil,
			wantErr:    "failed validating config: 1 error occurred:\n\t* unexpected Version 255 want 1\n\n",
		},
		{
			name: "test_invalid_signer_cache_timeout",
			cfg: `
signer_cache_timeout: -1m
`,
			wantConfig: nil,
			wantErr:    "failed validating config: 1 error occurred:\n\t* cache timeout invalid: -60000000000\n\n",
		},
		{
			name: "all_values_specified_env_override",
			cfg: `
version: 1
port: 8080
signer_cache_timeout: 1m
issuer: jvs
`,
			envs: map[string]string{
				"JVS_VERSION":              "1",
				"JVS_PORT":                 "tcp",
				"JVS_SIGNER_CACHE_TIMEOUT": "2m",
				"JVS_ISSUER":               "other",
			},
			wantConfig: &JustificationConfig{
				Version:            1,
				Port:               "tcp",
				SignerCacheTimeout: 2 * time.Minute,
				Issuer:             "other",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			lookuper := envconfig.MapLookuper(tc.envs)
			content := bytes.NewBufferString(tc.cfg).Bytes()
			gotConfig, err := loadJustificationConfigFromLookuper(ctx, content, lookuper)
			testutil.ErrCmp(t, tc.wantErr, err)
			if diff := cmp.Diff(tc.wantConfig, gotConfig); diff != "" {
				t.Errorf("Config unexpected diff (-want,+got):\n%s", diff)
			}
		})
	}
}
