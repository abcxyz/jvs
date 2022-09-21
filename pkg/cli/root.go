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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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
	rootCmd.PersistentFlags().String("server", "", "overwrite the JVS server address")
	rootCmd.PersistentFlags().Bool("insecure", false, "use insecure connection to JVS server")
	rootCmd.PersistentFlags().String("jwks_endpoint", "", "overwrite the JWKS endpoint")
	viper.BindPFlag("server", rootCmd.PersistentFlags().Lookup("server"))     //nolint // not expect err
	viper.BindPFlag("insecure", rootCmd.PersistentFlags().Lookup("insecure")) //nolint // not expect err
	viper.BindPFlag("jwks_endpoint", rootCmd.PersistentFlags().Lookup("jwks_endpoint"))     //nolint // not expect err

	rootCmd.AddCommand(tokenCmd)
	rootCmd.AddCommand(validateCmd)
}

func initCfg() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Try use the default config file.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(filepath.Join(home, ".jvsctl"))
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Also load from env vars.
	viper.SetEnvPrefix("JVSCTL")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// It's ok if the config file is not found because
		// the values could be filled by env vars or flags.
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			cobra.CheckErr(err)
			return
		}
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		cobra.CheckErr(err)
		return
	}

	if err := cfg.Validate(); err != nil {
		cobra.CheckErr(fmt.Errorf("invalid config: %w", err))
		return
	}
}

func ensureCfg(_ *cobra.Command, _ []string) error {
	if cfg == nil {
		return fmt.Errorf("CLI config missing or invalid")
	}
	return nil
}
