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
	"errors"
	"fmt"
	"os"

	"github.com/abcxyz/jvs/internal/version"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmdOptions are used as options to the root command.
type rootCmdOptions struct {
	configPath string
	version    bool
}

// newRootCmd creates a new instance of the root cobra command node.
func newRootCmd(cfg *config.CLIConfig) *cobra.Command {
	vpr := viper.New()

	opts := &rootCmdOptions{}

	cmd := &cobra.Command{
		Use:   "jvsctl",
		Short: "jvsctl facilitates the justification verification flow provided by abcxyz/jvs",

		// We bubble up errors in our main handler.
		SilenceErrors: true,

		// Usage is long and hides the error message.
		SilenceUsage: true,

		// Load the configuration before each child command. The actual "compiled"
		// configuration is passed in to the newChildCmd() function, but this sets
		// the values of that config.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return loadConfig(vpr, opts.configPath, cfg)
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.version {
				fmt.Fprintln(cmd.ErrOrStderr(), version.HumanVersion)
				return nil
			}
			return nil
		},
	}

	// Flags
	cmd.PersistentFlags().StringVarP(&opts.configPath, "config", "c", "",
		"path to a file on disk for the jvs configuration")
	cmd.PersistentFlags().StringVar(&cfg.Server, "server", "127.0.0.1:8080", "IP or DNS address to the JVS server")
	cmd.PersistentFlags().BoolVar(&cfg.Insecure, "insecure", false, "allow an insecure connection to the JVS server")
	cmd.PersistentFlags().StringVar(&cfg.JWKSEndpoint, "jwks_endpoint", "", "JWKS public key endpoint")
	cmd.Flags().BoolVarP(&opts.version, "version", "v", false, "print version and exit")

	// Subcommands
	cmd.AddCommand(newTokenCmd(cfg))
	cmd.AddCommand(newValidateCmd(cfg))

	return cmd
}

// loadConfig loads the configuration from the given path into the provided
// configuration struct.
func loadConfig(vpr *viper.Viper, pth string, cfg *config.CLIConfig) error {
	if pth == "" {
		pth = defaultConfigPath
	}

	vpr.SetConfigType("yaml")
	vpr.SetEnvPrefix("JVSCTL")
	vpr.AutomaticEnv()
	vpr.SetConfigFile(pth)

	if err := vpr.ReadInConfig(); err != nil {
		// Don't throw an error for failing to find the default config path since
		// it's not required. However, if the user specified a config path that
		// doesn't exist, that's an error.
		if !errors.Is(err, &viper.ConfigFileNotFoundError{}) && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to read configuration file: %w", err)
		}
		if pth != defaultConfigPath {
			return fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	if err := vpr.Unmarshal(cfg); err != nil {
		return fmt.Errorf("failed to unmarshal into config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}
	return nil
}
