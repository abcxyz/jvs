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
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abcxyz/jvs/pkg/config"
	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
)

func TestNewRootCmd(t *testing.T) {
	t.Parallel()

	configFile := filepath.Join(t.TempDir(), ".jvsctl.yaml")
	if err := os.WriteFile(configFile, []byte(strings.TrimSpace(`
server: 1.2.3.4:5678
insecure: true
	`)), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(configFile); err != nil {
			t.Error(err)
		}
	})

	cases := []struct {
		name      string
		args      []string
		expConfig *config.CLIConfig
	}{
		{
			name: "default_config",
			expConfig: &config.CLIConfig{
				Version: "1",
				Server:  "127.0.0.1:8080",
			},
		},
		{
			name: "flag_config",
			args: []string{"--config", configFile},
			expConfig: &config.CLIConfig{
				Version:  "1",
				Server:   "1.2.3.4:5678",
				Insecure: true,
			},
		},
		{
			name: "flag_server",
			args: []string{"--server", "1.2.3.4:5678"},
			expConfig: &config.CLIConfig{
				Version: "1",
				Server:  "1.2.3.4:5678",
			},
		},
		{
			name: "flag_insecure",
			args: []string{"--insecure"},
			expConfig: &config.CLIConfig{
				Version:  "1",
				Server:   "127.0.0.1:8080",
				Insecure: true,
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cfg config.CLIConfig
			cmd := newRootCmd(&cfg)

			// Add a fake subcommand to invoke for testing. This is required because
			// the pre/post hooks do not fire on root commands.
			cmd.AddCommand(&cobra.Command{
				Use: "testing",
				Run: func(cmd *cobra.Command, args []string) {},
			})

			args := append([]string{"testing"}, tc.args...)
			if _, _, err := testExecuteCommand(t, cmd, args...); err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.expConfig, &cfg); diff != "" {
				t.Errorf("config (-want, +got):\n%s", diff)
			}
		})
	}
}

// testExecteCommand executes the given cobra command and returns the stdout,
// stderr, and any execution error. See [testExecuteCommandStdin] for more
// information.
func testExecuteCommand(tb testing.TB, cmd *cobra.Command, args ...string) (string, string, error) {
	tb.Helper()
	return testExecuteCommandStdin(tb, cmd, nil, args...)
}

// testExecuteCommandStdin executes the given cobra command with the stdin
// reader w and returns the stdout, stderr, and any errors that occur during
// execution. It is safe for concurrent use if and only if the cobra command is
// safe for concurrent use (e.g. does not read or set global state). The
// convenience function [testExecuteCommand] exists for calling without stdin.
func testExecuteCommandStdin(tb testing.TB, cmd *cobra.Command, stdin io.Reader, args ...string) (string, string, error) {
	tb.Helper()

	var stdout, stderr bytes.Buffer
	defer stdout.Reset()
	defer stderr.Reset()

	cmd.SetIn(stdin)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}
