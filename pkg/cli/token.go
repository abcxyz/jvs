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
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	tokenExplanation string
	breakglass       bool
	ttl              time.Duration
)

var tokenCmd = &cobra.Command{
	Use:     "token",
	Short:   "To generate a justification token",
	Example: `token --explanation "issues/12345" -ttl 30m`,
	RunE:    runTokenCmd,
}

func runTokenCmd(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("not implemented")
}

func init() {
	tokenCmd.Flags().StringVarP(&tokenExplanation, "explanation", "e", "", "The explanation for the action")
	tokenCmd.MarkFlagRequired("explanation") //nolint // not expect err
	tokenCmd.Flags().BoolVar(&breakglass, "breakglass", false, "Whether it will be a breakglass action")
	tokenCmd.Flags().DurationVar(&ttl, "ttl", time.Hour, "The token time-to-live duration")
}
