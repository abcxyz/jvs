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

	"github.com/google/go-cmp/cmp"
	"github.com/sethvargo/go-envconfig"
)

func TestLoadPublicKeyConfig(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	envs := make(map[string]string)
	envs["PORT"] = "123"
	envs["CACHE_TIMEOUT"] = "5m"
	envs["KEY_RINGS"] = "key1,key2"

	lookuper := envconfig.MapLookuper(envs)
	content := bytes.NewBufferString("").Bytes()
	gotConfig, err := loadPublicKeyConfigFromLookuper(ctx, content, lookuper)
	if err != nil {
		t.Error(err)
	}

	wantConfig := &PublicKeyConfig{
		KeyRings:     []string{"key1", "key2"},
		CacheTimeout: 5 * time.Minute,
		Port:         "123",
	}
	if diff := cmp.Diff(wantConfig, gotConfig); diff != "" {
		t.Errorf("Config unexpected diff (-want,+got):\n%s", diff)
	}
}

func TestLoadPublicKeyConfig_Default(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	envs := make(map[string]string)
	envs["KEY_RINGS"] = "key1,key2"
	envs["CACHE_TIMEOUT"] = "5m"

	lookuper := envconfig.MapLookuper(envs)
	content := bytes.NewBufferString("").Bytes()
	gotConfig, err := loadPublicKeyConfigFromLookuper(ctx, content, lookuper)
	if err != nil {
		t.Error(err)
	}

	wantConfig := &PublicKeyConfig{
		KeyRings:     []string{"key1", "key2"},
		CacheTimeout: 5 * time.Minute,
		Port:         "8080",
	}
	if diff := cmp.Diff(wantConfig, gotConfig); diff != "" {
		t.Errorf("Config unexpected diff (-want,+got):\n%s", diff)
	}
}
