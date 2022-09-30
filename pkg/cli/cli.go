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

// Package cli implements the commands for the JVS CLI.
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/abcxyz/jvs/pkg/config"
)

const (
	// Issuer is the default issuer (iss) for tokens created by the CLI.
	Issuer = "jvsctl"

	// Subject is the default subject (sub) for tokens created by the CLI.
	Subject = "jvsctl"
)

// defaultConfigPath is the path on disk for the default configuration.
var defaultConfigPath = func() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".jvsctl", "config.yaml")
}()

// Execute executes the CLI.
func Execute(ctx context.Context) {
	var cfg config.CLIConfig

	cmd := newRootCmd(&cfg)
	if err := cmd.Execute(); err != nil {
		stderr := cmd.ErrOrStderr()
		fmt.Fprintln(stderr, err.Error())
		os.Exit(1)
	}
}
