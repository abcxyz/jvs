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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/client-lib/go/client"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/spf13/cobra"
)

// validateCmdOptions holds all the inputs and flags for the validate subcommand.
type validateCmdOptions struct {
	config *config.CLIConfig

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
Validate the given justification token passed via token flag or pipe.
The output will be a tablized view of validation result, token justification
and standard claims put together, or any errors that occured.

For example:

    # Validate the justification token string
    jvsctl validate --token "example token string"

    # Validate the justification token read from pipe
    cat token.txt | jvsctl token --token -

    # Output
	-------RESULT-------
	validated!
	breakglass  false
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
	flags.StringVarP(&opts.token, "token", "t", "",
		"The JVS token that needs validation")
	cmd.MarkFlagRequired("token") //nolint // not expect err

	return cmd
}

func runValidateCmd(cmd *cobra.Command, opts *validateCmdOptions, args []string) error {
	ctx := context.Background()

	jvsclient, err := client.NewJVSClient(ctx, &client.JVSConfig{
		Version:         "1",
		JWKSEndpoint:    opts.config.GetJWKSEndpoint(),
		CacheTimeout:    5 * time.Minute,
		AllowBreakglass: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create jvs client: %w", err)
	}

	if opts.token == "-" {
		stdin, ok := cmd.InOrStdin().(*os.File)
		// Default to os.Stdin when input is not of type os.File
		if !ok {
			stdin = os.Stdin
		}
		stat, _ := stdin.Stat()
		// Check whether the input is from pipe
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Read from pipe
			buf, err := io.ReadAll(io.LimitReader(stdin, 64*1_000))
			if err != nil {
				return fmt.Errorf("failed to read from pipe: %w", err)
			}
			opts.token = string(buf)
		} else {
			// Use user input
			fmt.Print("Enter token: ")
			fmt.Scanf("%s", &opts.token)
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
		token, err = jvsclient.ValidateJWT(opts.token)
		if err != nil {
			return fmt.Errorf("failed to validate jwt: %w", err)
		}
	}

	// Write validation result
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintf(w, "-------RESULT-------\nvalidated!\nbreakglass\t%t\n", breakglass); err != nil {
		return err
	}

	// Write justifications as a table
	if _, err := fmt.Fprintln(w, "----JUSTIFICATION----"); err != nil {
		return err
	}
	justs, err := jvspb.GetJustifications(token)
	if err != nil {
		return fmt.Errorf("failed to get justifications from token")
	}
	for _, j := range justs {
		if _, err := fmt.Fprintf(w, "%s\t\"%s\"\n", j.GetCategory(), j.GetValue()); err != nil {
			return err
		}
	}

	// Convert standard claims into a map, excluding justifications claim which is already handled
	claimsMap, err := token.AsMap(ctx)
	if err != nil {
		return fmt.Errorf("failed to convert token into map: %w", err)
	}
	delete(claimsMap, "justs")
	claims := make(map[string]string, len(claimsMap))
	claimsKeys := make([]string, 0, len(claimsMap))
	for k, v := range claimsMap {
		claimsKeys = append(claimsKeys, k)
		jv, err := json.Marshal(v)
		if err != nil {
			claims[k] = fmt.Sprintf("failed to marshal to json: %s", err)
		} else {
			claims[k] = string(jv)
		}
	}

	// Write standard claims in increasing order as a table
	sort.Strings(claimsKeys)
	if _, err := fmt.Fprintln(w, "---STANDARD CLAIMS---"); err != nil {
		return err
	}
	for _, k := range claimsKeys {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", k, claims[k]); err != nil {
			return err
		}
	}
	w.Flush()
	return nil
}
