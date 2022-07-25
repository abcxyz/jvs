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

func TestJustificationConfig_Defaults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var justificationConfig JustificationConfig
	if err := envconfig.ProcessWith(ctx, &justificationConfig, envconfig.MapLookuper(nil)); err != nil {
		t.Fatal(err)
	}

	want := &JustificationConfig{
		Version:            "1",
		Port:               "8080",
		SignerCacheTimeout: 5 * time.Minute,
		Issuer:             "jvs.abcxyz.dev",
	}

	if diff := cmp.Diff(want, &justificationConfig); diff != "" {
		t.Errorf("config with defaults (-want, +got):\n%s", diff)
	}
}

func TestLoadJustificationConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fakeFirestoreProjectID := "fakeProject"
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
firestore_project_id: fakeProject
signer_cache_timeout: 1m
issuer: jvs
`,
			wantConfig: &JustificationConfig{
				Port:               "123",
				Version:            "1",
				FirestoreProjectID: fakeFirestoreProjectID,
				SignerCacheTimeout: 1 * time.Minute,
				Issuer:             "jvs",
			},
		},
		{
			name: "test_default",
			cfg: `
firestore_project_id: fakeProject
`,
			wantConfig: &JustificationConfig{
				Port:               "8080",
				Version:            "1",
				FirestoreProjectID: fakeFirestoreProjectID,
				SignerCacheTimeout: 5 * time.Minute,
				Issuer:             "jvs.abcxyz.dev",
			},
		},
		{
			name: "test_wrong_version",
			cfg: `
version: 255
firestore_project_id: fakeProject
`,
			wantConfig: nil,
			wantErr:    `version "255" is invalid, valid versions are:`,
		},
		{
			name: "test_invalid_signer_cache_timeout",
			cfg: `
signer_cache_timeout: -1m
firestore_project_id: fakeProject
`,
			wantConfig: nil,
			wantErr:    `cache timeout must be a positive duration, got -1m0s`,
		},
		{
			name: "test_blank_project_id",
			cfg: `
port: 123
version: 1
signer_cache_timeout: 1m
issuer: jvs
`,
			wantConfig: nil,
			wantErr:    "firestore project id can't be empty",
		},
		{
			name: "all_values_specified_env_override",
			cfg: `
version: 1
port: 8080
firestore_project_id: fakeProject
signer_cache_timeout: 1m
issuer: jvs
`,
			envs: map[string]string{
				"JVS_VERSION":              "1",
				"JVS_PORT":                 "tcp",
				"JVS_FIRESTORE_PROJECT_ID": "fakeProject1",
				"JVS_SIGNER_CACHE_TIMEOUT": "2m",
				"JVS_ISSUER":               "other",
			},
			wantConfig: &JustificationConfig{
				Version:            "1",
				Port:               "tcp",
				FirestoreProjectID: "fakeProject1",
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
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if diff := cmp.Diff(tc.wantConfig, gotConfig); diff != "" {
				t.Errorf("Config unexpected diff (-want,+got):\n%s", diff)
			}
		})
	}
}
