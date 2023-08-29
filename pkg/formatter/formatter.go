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

// Package formatter exposes printers and formatters for justifications and
// justification tokens.
package formatter

import (
	"context"
	"fmt"
	"io"
	"sort"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

// Formatter is an interface which all formatters must implement.
type Formatter interface {
	// FormatTo streams the result to the writer.
	FormatTo(ctx context.Context, w io.Writer, token jwt.Token, breakglass bool) error
}

// structure is the internal structure for printing json and yaml.
type structure struct {
	// Breakglass indicates whether this is a breakglass token.
	Breakglass bool `json:"breakglass" yaml:"breakglass"`

	// Justifications is the list of justifications.
	Justifications []*justification `json:"justifications" yaml:"justifications"`

	// Claims is the list of claims.
	Claims map[string]any `json:"claims" yaml:"claims"`
}

// justification is the internal representation so we can add json and yaml
// annotations.
type justification struct {
	// Category is the justification category.
	Category string `json:"category" yaml:"category"`

	// Value is the justification value.
	Value string `json:"value" yaml:"value"`

	// Annotation stores additional info the plugin may want to encapsulate in the Justification.
	// It's not intended for user input.
	Annotation map[string]string `json:"annotation" yaml:"annotation"`
}

// toStructure creates our internal structure from the token.
func toStructure(ctx context.Context, token jwt.Token, breakglass bool) (*structure, error) {
	var s structure
	s.Breakglass = breakglass

	// Write justifications
	justifications, err := jvspb.GetJustifications(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get justifications from token: %w", err)
	}
	s.Justifications = make([]*justification, 0, len(justifications))
	for _, j := range justifications {
		s.Justifications = append(s.Justifications, &justification{
			Category:   j.GetCategory(),
			Value:      j.GetValue(),
			Annotation: j.GetAnnotation(),
		})
	}
	sort.Slice(s.Justifications, func(i, j int) bool {
		return s.Justifications[i].Category < s.Justifications[j].Category
	})

	// Write other claims
	standard, err := token.AsMap(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to convert token claims into map: %w", err)
	}
	delete(standard, jvspb.JustificationsKey)
	s.Claims = standard

	return &s, nil
}
