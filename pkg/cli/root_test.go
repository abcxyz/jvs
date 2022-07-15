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

package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/abcxyz/jvs/pkg/config"
	"github.com/google/go-cmp/cmp"
)

func TestInitCfg(t *testing.T) {
	cfgFile = filepath.Join(t.TempDir(), ".jvsctl.yaml")

	if err := os.WriteFile(cfgFile, []byte(`server: https://example.com
`), fs.ModePerm); err != nil {
		t.Fatalf("failed to prepare test config file: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Remove(cfgFile); err != nil {
			t.Logf("failed to cleanup test config file: %v", err)
		}
	})

	initCfg()

	wantCfg := &config.CLIConfig{
		Version:        "1",
		Server:         "https://example.com",
		Authentication: &config.CLIAuthentication{},
	}
	if diff := cmp.Diff(wantCfg, cfg); diff != "" {
		t.Errorf("CLI config loaded (-want,+got):\n%s", diff)
	}
}
