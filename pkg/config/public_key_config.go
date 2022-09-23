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
	"time"
)

// PublicKeyConfig is the config used for public key hosting.
type PublicKeyConfig struct {
	// TODO: This is intended to be temporary, and will eventually be retrieved from a persistent external datastore
	// https://github.com/abcxyz/jvs/issues/17
	// KeyName format: `projects/*/locations/*/keyRings/*/cryptoKeys/*`
	// https://pkg.go.dev/google.golang.org/genproto/googleapis/cloud/kms/v1#PublicKeyKey
	KeyNames     []string      `yaml:"key_names,omitempty" env:"KEY_NAMES,overwrite"`
	CacheTimeout time.Duration `yaml:"cache_timeout" env:"CACHE_TIMEOUT"`
	Port         string        `env:"PORT,default=8080"`
}
