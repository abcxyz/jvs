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
	"fmt"
	"io"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"gopkg.in/yaml.v3"
)

// YAML outputs the JWT as a pretty-printed YAML blob.
type YAML struct{}

// Ensure YAML implements Formatter.
var _ Formatter = (*YAML)(nil)

// NewYAML creates a new text formatter.
func NewYAML() *YAML {
	return &YAML{}
}

// FormatTo renders the token to the given writer as yaml.
func (y *YAML) FormatTo(ctx context.Context, w io.Writer, token jwt.Token, breakglass bool) error {
	s, err := toStructure(ctx, token, breakglass)
	if err != nil {
		return err
	}

	// Encode
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(s); err != nil {
		return fmt.Errorf("failed to encode to yaml: %w", err)
	}

	if err := enc.Close(); err != nil {
		return fmt.Errorf("failed to close yaml encoder: %w", err)
	}

	return nil
}
