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
)

func TestValidateCLIConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *CLIConfig
		wantErr string
	}{{
		name: "no_error",
		cfg:  &CLIConfig{Server: "example.com"},
	}, {
		name:    "missing_server_error",
		cfg:     &CLIConfig{},
		wantErr: "missing JVS server address",
	}}

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