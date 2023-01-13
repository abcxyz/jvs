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
	"github.com/mitchellh/mapstructure"
)

const (
	// JustificationsKey is the key in the JWT where justifications are stored.
	// Ideally this would be "justifications", but the RFC and various online
	// resources recommend key names be as short as possible to keep the JWTs
	// small. Akamai recommends less than 8 characters and Okta recommends less
	// than 6.
	//
	// Most callers should use the higher-level functions, but this is exposed in
	// case users need to manipulate lower-level structures in the claims map.
	JustificationsKey string = "justs"
)

// WithTypedJustifications is an option for parsing JWTs that will convert
// decode the [Justification] claims into the correct Go structure. If this is
// not supplied, the claims will be "any" and future type assertions may fail.
func WithTypedJustifications() jwt.ParseOption {
	return jwt.WithTypedClaim(JustificationsKey, []*Justification{})
}

// GetJustifications retrieves a copy of the justifications on the token. If the
// token does not have any justifications, it returns an empty slice of
// justifications.
//
// This function is incredibly defensive against a poorly-parsed jwt. It handles
// situations where the JWT was not properly decoded (i.e. the caller did not
// use [WithTypedJustifications]), and when the token uses a single
// justification instead of a slice.
//
// Modifying the slice does not modify the underlying token - you must call
// [SetJustifications] to update the data on the token.
func GetJustifications(t jwt.Token) ([]*Justification, error) {
	if t == nil {
		return nil, fmt.Errorf("token cannot be nil")
	}

	raw, ok := t.Get(JustificationsKey)
	if !ok {
		return []*Justification{}, nil
	}

	var claims []*Justification
	switch list := raw.(type) {
	case []*Justification:
		// Token was decoded with typed claims.
		claims = list
	case *Justification:
		// Token did not provide a list.
		claims = []*Justification{list}
	case []any:
		// Token was a proto but wasn't decoded.
		if err := mapstructure.Decode(list, &claims); err != nil {
			return nil, fmt.Errorf("found justifications, but could not decode map data: %w", err)
		}
	default:
		return nil, fmt.Errorf("found justifications, but was of unknown type %T", raw)
	}

	// Make a copy of the slice so we don't modify the underlying data structure.
	cp := make([]*Justification, 0, len(claims))
	cp = append(cp, claims...)
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
	if err := t.Set(JustificationsKey, cp); err != nil {
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

	if err := t.Remove(JustificationsKey); err != nil {
		return fmt.Errorf("failed to remove justifications: %w", err)
	}
	return nil
}
