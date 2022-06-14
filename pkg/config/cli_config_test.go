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

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-envconfig"
)

func TestLoadCLIConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name       string
		cfg        string
		envs       map[string]string
		wantConfig *CLIConfig
		wantErr    string
	}{{
		name: "all_values_specified_in_file",
		cfg: `version: 1
server: https://example.com
`,
		wantConfig: &CLIConfig{
			Version: 1,
			Server:  "https://example.com",
		},
	}, {
		name: "server_overwritten_with_env_var",
		cfg: `version: 1
`,
		envs: map[string]string{
			"JVSCTL_SERVER": "https://example.com",
		},
		wantConfig: &CLIConfig{
			Version: 1,
			Server:  "https://example.com",
		},
	}, {
		name: "missing_server_error",
		cfg: `version: 1
`,
		wantErr: "missing JVS server address",
	}}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			lookuper := envconfig.MapLookuper(tc.envs)
			content := bytes.NewBufferString(tc.cfg).Bytes()
			gotConfig, err := loadCLIConfigFromLookuper(ctx, content, lookuper)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected err: %s", diff)
			}
			if diff := cmp.Diff(tc.wantConfig, gotConfig); diff != "" {
				t.Errorf("config unexpected diff (-want,+got):\n%s", diff)
			}
		})
	}
}
