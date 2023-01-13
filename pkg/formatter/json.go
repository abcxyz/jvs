// Copyright 2023 Google LLC
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

package formatter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/lestrrat-go/jwx/v2/jwt"
)

// JSON outputs the JWT as a pretty-printed JSON blob.
type JSON struct{}

// Ensure JSON implements Formatter.
var _ Formatter = (*JSON)(nil)

// NewJSON creates a new text formatter.
func NewJSON() *JSON {
	return &JSON{}
}

// FormatTo renders the token to the given writer as json.
func (j *JSON) FormatTo(ctx context.Context, w io.Writer, token jwt.Token, breakglass bool) error {
	s, err := toStructure(ctx, token, breakglass)
	if err != nil {
		return err
	}

	// Encode
	if err := json.NewEncoder(w).Encode(s); err != nil {
		return fmt.Errorf("failed to encode to json: %w", err)
	}
	return nil
}
