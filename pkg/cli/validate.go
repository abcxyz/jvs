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
	"text/tabwriter"
	"time"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/client-lib/go/client"
	"github.com/spf13/cobra"
)

var (
	flagToken string
	stdin = os.Stdin
)

var validateCmd = &cobra.Command{
	Use:     "validate",
	Short:   "To validate the given justification token",
	Example: `validate --token "example token"`,
	RunE:    runValidateCmd,
}

func runValidateCmd(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	jvsclient, err := client.NewJVSClient(ctx, &client.JVSConfig{
		Version:         "1",
		JWKSEndpoint:    cfg.JWKSEndpoint,
		CacheTimeout:    5 * time.Minute,
		AllowBreakglass: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create jvs client: %w", err)
	}

	if flagToken == "-" {
		stat, _ := stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Read from pipe
			buf, err := io.ReadAll(io.LimitReader(stdin, 64*1_000))
			if err != nil {
				return fmt.Errorf("failed to read from pipe: %w", err)
			}
			flagToken = string(buf)
		} else {
			// User input
			fmt.Print("Enter token: ")
			fmt.Scanf("%s", &flagToken)
		}
	}

	// Validate token
	breakglass := false
	token, err := jvspb.ParseBreakglassToken(flagToken)
	if err != nil {
		return fmt.Errorf("failed parse breakglass token: %w", err)
	}
	if token != nil {
		breakglass = true
	} else {
		token, err = jvsclient.ValidateJWT(flagToken)
		if err != nil {
			return fmt.Errorf("failed validate jwt: %w", err)
		}
	}

	// Convert parsed token to map
	claimsMap, err := token.AsMap(ctx)
	if err != nil {
		return fmt.Errorf("failed convert token into map: %w", err)
	}
	claimsMap["breakglass"] = breakglass
	claimsMap["valid"] = true

	claimsKeys := make([]string, 0, len(claimsMap))
	claims := make(map[string]string, len(claimsMap))
	for k, v := range claimsMap {
		claimsKeys = append(claimsKeys, k)
		jv, err := json.Marshal(v)
		if err != nil {
			claims[k] = fmt.Sprintf("failed to marshal to json: %s", err)
		} else {
			claims[k] = string(jv)
		}
	}
	sort.Strings(claimsKeys)

	// Output token claims into a table
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 1, ' ', 0)
	for _, k := range claimsKeys {
		if _, err := fmt.Fprint(w, fmt.Sprintf("%s\t%s", k, claims[k])); err != nil {
			return err
		}
		fmt.Fprintln(w)
	}
	w.Flush()
	return nil
}

func init() {
	validateCmd.Flags().StringVarP(&flagToken, "token", "t", "", "The token that needs validation")
	validateCmd.MarkFlagRequired("token") //nolint // not expect err
}
