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
	"io"
	"strings"
	"time"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/client-lib/go/client"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/formatter"
	"github.com/spf13/cobra"
)

// cacheTimeout is required for creating jvs client via jvs config, it is not really used since cache is expired when CLI exits.
const cacheTimeout = 5 * time.Minute

// validateCmdOptions holds all the inputs and flags for the validate subcommand.
type validateCmdOptions struct {
	config *config.CLIConfig

	// format flag to the command.
	format string

	// token flag to the command.
	token string
}

// newValidateCmd creates a new subcommand for validating tokens.
func newValidateCmd(cfg *config.CLIConfig) *cobra.Command {
	opts := &validateCmdOptions{
		config: cfg,
	}

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate the input token",
		Long: strings.Trim(`
Validate the given justification token and output the justifications and other
standard claims if it's valid. If the token is invalid, an error will be returned.

For example:

    # Validate the justification token string
    jvsctl validate --token "example token string"

    # Validate the justification token read from pipe
    cat token.txt | jvsctl validate --token -

    # Output
    ------BREAKGLASS------
    false

    ----JUSTIFICATION----
    explanation  "test"

    ---STANDARD CLAIMS---
    aud  ["dev.abcxyz.jvs"]
    iat  "2022-01-01T00:00:00Z"
    iss  "jvsctl"
    jti  "test-jwt"
    nbf  "2022-01-01T00:00:00Z"
    sub  "jvsctl"
`, "\n"),
		Args: cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidateCmd(cmd, opts, args)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opts.format, "format", "f", "table",
		"output format (valid values: table, json, yaml)")
	flags.StringVarP(&opts.token, "token", "t", "",
		"JVS token that needs validation, can be passed as string or via pipe")
	cmd.MarkFlagRequired("token") //nolint // not expect err
	return cmd
}

func runValidateCmd(cmd *cobra.Command, opts *validateCmdOptions, args []string) error {
	ctx := context.Background()

	jvsclient, err := client.NewJVSClient(ctx, &client.JVSConfig{
		Version:         "1",
		JWKSEndpoint:    opts.config.JWKSEndpoint,
		CacheTimeout:    cacheTimeout,
		AllowBreakglass: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create jvs client: %w", err)
	}

	// Compute the formatter
	var f formatter.Formatter
	switch v := strings.TrimSpace(strings.ToLower(opts.format)); v {
	case "json":
		f = formatter.NewJSON()
	case "", "table", "text":
		f = formatter.NewText()
	case "yaml":
		f = formatter.NewYAML()
	default:
		return fmt.Errorf("unknown formatter %q", v)
	}

	// Read token from stdin
	if opts.token == "-" {
		buf, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), 64*1_000))
		if err != nil || len(buf) == 0 {
			fmt.Print("Enter token: ")
			fmt.Scanf("%s", &opts.token)
		} else {
			opts.token = string(buf)
		}
	}

	// Validate token
	breakglass := false
	token, err := jvspb.ParseBreakglassToken(opts.token)
	if err != nil {
		return fmt.Errorf("failed to parse breakglass token: %w", err)
	}
	if token != nil {
		breakglass = true
	} else {
		token, err = jvsclient.ValidateJWT(ctx, opts.token)
		if err != nil {
			return fmt.Errorf("failed to validate jwt: %w", err)
		}
	}

	if err := f.FormatTo(ctx, cmd.OutOrStdout(), token, breakglass); err != nil {
		return fmt.Errorf("failed to format token: %w", err)
	}
	return nil
}
