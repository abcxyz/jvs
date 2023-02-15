// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"testing"

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-envconfig"
)

func TestNewConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	tests := []struct {
		name       string
		envs       map[string]string
		wantConfig *UIServiceConfig
		wantErr    string
	}{
		{
			name: "default_port",
			envs: map[string]string{
				"ALLOW_LIST": "example.com",
			},
			wantConfig: &UIServiceConfig{
				Port:      "9091",
				AllowList: []string{"example.com"},
			},
		},
		{
			name: "override_port",
			envs: map[string]string{
				"PORT":       "1010",
				"ALLOW_LIST": "example.com",
			},
			wantConfig: &UIServiceConfig{
				Port:      "1010",
				AllowList: []string{"example.com"},
			},
		},
		{
			name:       "no_port_no_allow_list",
			envs:       map[string]string{},
			wantConfig: nil,
			wantErr:    "failed to parse server config:",
		},
		{
			name: "asterisks_in_allow_list",
			envs: map[string]string{
				"ALLOW_LIST": "example.com;*",
			},
			wantConfig: nil,
			wantErr:    "asterisk(*) must be exclusive, no other domains allowed",
		},
		{
			name: "exclusive_asterisk",
			envs: map[string]string{
				"ALLOW_LIST": "*",
			},
			wantConfig: &UIServiceConfig{
				Port:      "9091",
				AllowList: []string{"*"},
			},
		},
		{
			name: "multiple_domains",
			envs: map[string]string{
				"ALLOW_LIST": "subdomain.foo.com;*.example.com",
			},
			wantConfig: &UIServiceConfig{
				Port:      "9091",
				AllowList: []string{"subdomain.foo.com", "*.example.com"},
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			lookuper := envconfig.MapLookuper(tc.envs)
			gotConfig, err := newUIConfig(ctx, lookuper)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if diff := cmp.Diff(tc.wantConfig, gotConfig); diff != "" {
				t.Errorf("Config unexpected diff (-want,+got):\n%s", diff)
			}
		})
	}
}
