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
	"time"

	"github.com/abcxyz/jvs/client-lib/go/client"
	"github.com/spf13/cobra"
)

var (
	flagToken           string
	flagAllowBreakglass bool
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
		AllowBreakglass: flagAllowBreakglass,
	})
	if err != nil {
		return err
	}

	_, err = jvsclient.ValidateJWT(flagToken)
	if err != nil {
		return err
	}

	_, err = cmd.OutOrStdout().Write([]byte("Token is valid"))
	return err
}

func init() {
	validateCmd.Flags().StringVarP(&flagToken, "token", "t", "", "The token that needs validation")
	validateCmd.MarkFlagRequired("token") //nolint // not expect err
	validateCmd.Flags().BoolVar(&flagAllowBreakglass, "allow_breakglass", false, "Whether breakglass is allowed")
}
