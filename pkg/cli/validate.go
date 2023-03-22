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
	"context"
	"fmt"
	"strings"
	"time"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/client-lib/go/client"
	"github.com/abcxyz/jvs/pkg/formatter"
	"github.com/abcxyz/pkg/cli"
)

// cacheTimeout is required for creating jvs client via jvs config, it is not really used since cache is expired when CLI exits.
const cacheTimeout = 5 * time.Minute

var _ cli.Command = (*ValidateCommand)(nil)

type ValidateCommand struct {
	cli.BaseCommand

	flagToken        string
	flagSubject      string
	flagJWKSEndpoint string
	flagFormat       string
}

func (c *ValidateCommand) Desc() string {
	return `Validate the input token`
}

func (c *ValidateCommand) Help() string {
	return strings.Trim(`
Validate the given justification token and output the justifications and other
standard claims if it's valid. If the token is invalid, an error will be returned.

EXAMPLES

    # Validate the justification token string
    jvsctl validate -token "example token string"

    # Validate the justification token read from pipe
    cat token.txt | jvsctl validate -token -

`+c.Flags().Help(), "\n")
}

func (c *ValidateCommand) Flags() *cli.FlagSet {
	set := cli.NewFlagSet()

	// Command options
	f := set.NewSection("COMMAND OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "token",
		Target:  &c.flagToken,
		Example: "ya29.c...",
		Usage: `The JVS token that needs to be validated. Set the value to ` +
			`"-" to read from stdin.`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "subject",
		Target:  &c.flagSubject,
		Example: "you@example.com",
		Usage:   `The subject to validate in the token.`,
	})

	f.StringVar(&cli.StringVar{
		Name:    "format",
		Aliases: []string{"f"},
		Target:  &c.flagFormat,
		Example: "table",
		Default: "table",
		Usage:   `The target output format. Valid values are: table, json, yaml.`,
	})

	// Server flags
	f = set.NewSection("SERVER OPTIONS")

	f.StringVar(&cli.StringVar{
		Name:    "jwks-endpoint",
		Target:  &c.flagJWKSEndpoint,
		Example: "https://jvs.example.com:8080/.well-known/jwks",
		Default: "http://localhost:8080/.well-known/jwks",
		EnvVar:  "JVSCTL_JWKS_ENDPOINT",
		Usage: `JVS public key server endpoint including the protocol, ` +
			`address, port, and .well-known path.`,
	})

	return set
}

func (c *ValidateCommand) Run(ctx context.Context, args []string) error {
	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}
	args = f.Args()
	if len(args) > 0 {
		return fmt.Errorf("unexpected arguments: %q", args)
	}

	if c.flagToken == "" {
		return fmt.Errorf("token is required")
	}

	// Compute the formatter
	var format formatter.Formatter
	switch v := strings.TrimSpace(strings.ToLower(c.flagFormat)); v {
	case "json":
		format = formatter.NewJSON()
	case "", "table", "text":
		format = formatter.NewText()
	case "yaml":
		format = formatter.NewYAML()
	default:
		return fmt.Errorf("unknown formatter %q", v)
	}

	// Read token from stdin
	if c.flagToken == "-" {
		token, err := c.Prompt("Enter token: ")
		if err != nil {
			return fmt.Errorf("failed to get token from prompt: %w", err)
		}
		c.flagToken = token
	}

	// Validate token
	breakglass := false
	token, err := jvspb.ParseBreakglassToken(ctx, c.flagToken)
	if err != nil {
		return fmt.Errorf("failed to parse breakglass token: %w", err)
	}
	if token != nil {
		breakglass = true
	} else {
		jvsclient, err := client.NewJVSClient(ctx, &client.JVSConfig{
			Version:         "1",
			JWKSEndpoint:    c.flagJWKSEndpoint,
			CacheTimeout:    cacheTimeout,
			AllowBreakglass: true,
		})
		if err != nil {
			return fmt.Errorf("failed to create jvs client: %w", err)
		}

		token, err = jvsclient.ValidateJWT(ctx, c.flagToken, c.flagSubject)
		if err != nil {
			return fmt.Errorf("failed to validate jwt: %w", err)
		}
	}

	if err := format.FormatTo(ctx, c.Stdout(), token, breakglass); err != nil {
		return fmt.Errorf("failed to format token: %w", err)
	}
	return nil
}
