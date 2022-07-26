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

// Package util provides utilities that are intended to enable easier
// and more concise writing of source code.
package util

import (
	"context"
	"fmt"

	"github.com/abcxyz/jvs/pkg/config"
)

const (
	keyNameField = "key_name"
)

// GetKeyNames fetch key names from Key Config.
func GetKeyNames(ctx context.Context, keyCfg config.RemoteConfig) ([]string, error) {
	kmsKeyNames := make([]string, 0)
	v, err := keyCfg.Get(ctx, keyNameField)
	if err != nil {
		return kmsKeyNames, fmt.Errorf("failed when getting key name from key config %w", err)
	}
	keyName, stringOk := v.(string)
	keyNames, stringArrayOk := v.([]string)
	if stringOk && stringArrayOk {
		return kmsKeyNames, fmt.Errorf("invalid remote config field which stores kms keys")
	}
	if stringOk {
		kmsKeyNames = append(kmsKeyNames, keyName)
	}
	if stringArrayOk {
		kmsKeyNames = append(kmsKeyNames, keyNames...)
	}
	return kmsKeyNames, nil
}
