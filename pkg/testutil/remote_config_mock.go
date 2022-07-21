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

// Package testutil provides utilities that are intended to enable easier
// and more concise writing of unit test code.
package testutil

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type MockRemoteConfig struct {
	fileName string
	v        *viper.Viper
}

func NewMockRemoteConfig(jsonStr, fileName string) (*MockRemoteConfig, error) {
	v := viper.New()
	fs := afero.NewMemMapFs()
	v.SetFs(fs)
	v.SetConfigName(fileName)
	v.SetConfigType("json")
	if err := v.ReadConfig(bytes.NewBuffer([]byte(jsonStr))); err != nil {
		return nil, err
	}
	if err := v.WriteConfigAs(fileName + ".json"); err != nil {
		return nil, err
	}
	return &MockRemoteConfig{fileName: fileName, v: v}, nil
}

func (m MockRemoteConfig) Load(ctx context.Context, data any) error {
	return m.v.Unmarshal(data)
}

func (m MockRemoteConfig) GetByKey(ctx context.Context, key string) (any, error) {
	return m.v.Get(key), nil
}

func (m MockRemoteConfig) SetByKey(ctx context.Context, key string, value any) error {
	if key == "" {
		vBytes, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if err := m.v.ReadConfig(bytes.NewBuffer(vBytes)); err != nil {
			return err
		}
		if err := m.v.WriteConfigAs(m.fileName + ".json"); err != nil {
			return err
		}
	} else {
		m.v.Set(key, value)
	}
	return nil
}
