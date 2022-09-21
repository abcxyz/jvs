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

package v0

import (
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	// jwtJustificationsKey is the key in the JWT where justifications are stored.
	// Ideally this would be "justifications", but the RFC and various online
	// resources recommend key names be as short as possible to keep the JWTs
	// small. Akamai recommends less than 8 characters and Okta recommends less
	// than 6.
	jwtJustificationsKey string = "justs"
)

// WithTypedJustifications is an option for parsing JWTs that will convert
// decode the [Justification] claims into the correct Go structure. If this is
// not supplied, the claims will be "any" and future type assertions will fail.
func WithTypedJustifications() jwt.ParseOption {
	return jwt.WithTypedClaim(jwtJustificationsKey, []*Justification{})
}

// GetJustifications retrieves a copy of the justifications on the token. If the
// token does not have any justifications, it returns an empty slice of
// justifications. Modifying the slice does not modify the underlying token -
// you must call SetJustifications to update the data on the token.
func GetJustifications(t jwt.Token) ([]*Justification, error) {
	if t == nil {
		return nil, fmt.Errorf("token cannot be nil")
	}

	raw, ok := t.Get(jwtJustificationsKey)
	if !ok {
		return []*Justification{}, nil
	}

	typ, ok := raw.([]*Justification)
	if !ok {
		return nil, fmt.Errorf("found justifications, but was %T (expected %T)",
			raw, []*Justification{})
	}

	// Make a copy of the slice so we don't modify the underlying data structure.
	cp := make([]*Justification, 0, len(typ))
	cp = append(cp, typ...)
	return cp, nil
}

// SetJustifications updates the justifications on the token. It overwrites any
// existing values and uses a copy of the inbound slice.
func SetJustifications(t jwt.Token, justifications []*Justification) error {
	if t == nil {
		return fmt.Errorf("token cannot be nil")
	}

	cp := make([]*Justification, 0, len(justifications))
	cp = append(cp, justifications...)
	return t.Set(jwtJustificationsKey, cp)
}

// AppendJustification appends the given justification to the end of the current
// justifications list. It does not check for duplication and does not lock the
// token. There is a possible race between when the claims are read and when the
// claims are set back on the token. Callers should use [SetJustifications]
// directly to avoid this race.
func AppendJustification(t jwt.Token, justification *Justification) error {
	if t == nil {
		return fmt.Errorf("token cannot be nil")
	}

	justifications, err := GetJustifications(t)
	if err != nil {
		return fmt.Errorf("failed to get justifications: %w", err)
	}

	justifications = append(justifications, justification)
	if err := SetJustifications(t, justifications); err != nil {
		return fmt.Errorf("failed to set justifications: %w", err)
	}
	return nil
}

// ClearJustifications removes the justifications from the token by deleting the
// entire key.
func ClearJustifications(t jwt.Token) error {
	if t == nil {
		return fmt.Errorf("token cannot be nil")
	}

	return t.Remove(jwtJustificationsKey)
}
