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
	"context"
	"testing"
	"time"

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-envconfig"
)

func TestCertRotationConfig_Defaults(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var cfg CertRotationConfig
	if err := envconfig.ProcessWith(ctx, &cfg, envconfig.MapLookuper(nil)); err != nil {
		t.Fatal(err)
	}

	want := &CertRotationConfig{
		DevMode: false,
		Port:    "8080",
		CryptoConfig: &CryptoConfig{
			Version: "1",
		},
	}

	if diff := cmp.Diff(want, &cfg); diff != "" {
		t.Errorf("config with defaults (-want, +got):\n%s", diff)
	}
}

func TestCertRotationConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("validates_crypto_config", func(t *testing.T) {
		t.Parallel()

		cfg := &CertRotationConfig{
			CryptoConfig: &CryptoConfig{
				GracePeriod: -1 * time.Minute, // must be positive, should fail validation
			},
		}

		err := cfg.Validate()
		if diff := testutil.DiffErrString(err, "grace period must be a positive duration"); diff != "" {
			t.Errorf(diff)
		}
	})
}
