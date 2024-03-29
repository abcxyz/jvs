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

	"github.com/abcxyz/jvs/internal/version"
	"github.com/abcxyz/pkg/cli"
)

const (
	// Issuer is the default issuer (iss) for tokens created by the CLI.
	Issuer = "jvsctl"
)

// rootCmd defines the starting command structure.
var rootCmd = func() cli.Command {
	return &cli.RootCommand{
		Name:    "jvsctl",
		Version: version.HumanVersion,
		Commands: map[string]cli.CommandFactory{
			"api": func() cli.Command {
				return &cli.RootCommand{
					Name:        "api",
					Description: "Perform API operations",
					Commands: map[string]cli.CommandFactory{
						"server": func() cli.Command {
							return &APIServerCommand{}
						},
					},
				}
			},
			"public-key": func() cli.Command {
				return &cli.RootCommand{
					Name:        "public-key",
					Description: "Perform public-key operations",
					Commands: map[string]cli.CommandFactory{
						"server": func() cli.Command {
							return &PublicKeyServerCommand{}
						},
					},
				}
			},
			"rotation": func() cli.Command {
				return &cli.RootCommand{
					Name:        "rotation",
					Description: "Perform rotation operations",
					Commands: map[string]cli.CommandFactory{
						"server": func() cli.Command {
							return &RotationServerCommand{}
						},
					},
				}
			},
			"token": func() cli.Command {
				return &cli.RootCommand{
					Name:        "token",
					Description: "Perform token operations",
					Commands: map[string]cli.CommandFactory{
						"create": func() cli.Command {
							return &TokenCreateCommand{}
						},
						"validate": func() cli.Command {
							return &TokenValidateCommand{}
						},
					},
				}
			},
			"ui": func() cli.Command {
				return &cli.RootCommand{
					Name:        "ui",
					Description: "Perform ui operations",
					Commands: map[string]cli.CommandFactory{
						"server": func() cli.Command {
							return &UIServerCommand{}
						},
					},
				}
			},
		},
	}
}

// Run executes the CLI.
func Run(ctx context.Context, args []string) error {
	return rootCmd().Run(ctx, args) //nolint:wrapcheck // Want passthrough
}
