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

	"github.com/spf13/cobra"

	"github.com/abcxyz/jvs/pkg/config"
)

var (
	cfgFile string
	cfg     *config.CLIConfig
)

var rootCmd = &cobra.Command{
	Use:               "jvsctl",
	Short:             "jvsctl facilitates the justification verification flow provided by abcxyz/jvs",
	PersistentPreRunE: ensureCfg,
}

// Execute executes the CLI.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initCfg)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jvsctl/config.yaml)")

	rootCmd.AddCommand(tokenCmd)
}

func initCfg() {
	if cfgFile == "" {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			cobra.CheckErr(err)
			return
		}
		cfgFile = filepath.Join(home, ".jvsctl", "config.yaml")
	}

	f, err := os.ReadFile(cfgFile)
	if err != nil {
		cobra.CheckErr(err)
		return
	}

	c, err := config.LoadCLIConfig(context.Background(), f)
	if err != nil {
		cobra.CheckErr(err)
		return
	}

	cfg = c
}

func ensureCfg(_ *cobra.Command, _ []string) error {
	if cfg == nil {
		return fmt.Errorf("CLI config missing")
	}
	return nil
}
