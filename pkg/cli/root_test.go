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
	"strings"
	"testing"
)

func TestRootCommand_Help(t *testing.T) {
	t.Parallel()

	exp := `
Usage: jvsctl COMMAND

  api           Perform API operations
  public-key    Perform public-key operations
  rotation      Perform rotation operations
  token         Generate a justification token
  ui            Perform ui operations
  validate      Validate the input token
`

	cmd := rootCmd()
	if got, want := strings.TrimSpace(cmd.Help()), strings.TrimSpace(exp); got != want {
		t.Errorf("expected\n\n%s\n\nto be\n\n%s\n\n", got, want)
	}
}
