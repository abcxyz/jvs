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
	"testing"

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestCreateBreakglassToken(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		token jwt.Token
		err   string
	}{
		{
			name:  "nil",
			token: nil,
			err:   "cannot be nil",
		},
		{
			name:  "nil",
			token: testTokenBuilder(t, jwt.NewBuilder()),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tokenStr, err := CreateBreakglassToken(tc.token, "testing")
			if diff := testutil.DiffErrString(err, tc.err); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if err != nil {
				return
			}

			// Parse the token and verify the justifications.
			parsed, err := jwt.ParseString(tokenStr,
				jwt.WithKey(jwa.HS256, []byte(BreakglassHMACSecret)))
			if err != nil {
				t.Fatal(err)
			}

			justifications, err := GetJustifications(parsed)
			if err != nil {
				t.Fatal(err)
			}
			expectedJustifications := []*Justification{
				// This intentionally doesn't use the constants as a reminder not to
				// change the constants.
				{
					Category: "breakglass",
					Value:    "testing",
				},
			}
			if diff := cmp.Diff(expectedJustifications, justifications, cmpopts.IgnoreUnexported(Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestParseBreakglassToken(t *testing.T) {
	t.Parallel()

	breakglassToken := testTokenBuilder(t, jwt.NewBuilder())
	if err := SetJustifications(breakglassToken, []*Justification{
		{
			Category: breakglassJustificationCategory,
			Value:    "testing",
		},
	}); err != nil {
		t.Fatal(err)
	}

	tokenWithoutJustifications := testTokenBuilder(t, jwt.NewBuilder())

	cases := []struct {
		name     string
		tokenStr string
		err      string
	}{
		{
			name:     "empty_string",
			tokenStr: "",
			err:      "failed to parse token headers",
		},
		{
			name:     "invalid_algorithm",
			tokenStr: testSignToken(t, breakglassToken, jwt.WithKey(jwa.HS512, []byte(BreakglassHMACSecret))),
		},
		{
			name:     "invalid_signature",
			tokenStr: testSignToken(t, breakglassToken, jwt.WithKey(jwa.HS256, []byte("foobar"))),
			err:      "could not verify message",
		},
		{
			name:     "missing_justifications",
			tokenStr: testSignToken(t, tokenWithoutJustifications, jwt.WithKey(jwa.HS256, []byte(BreakglassHMACSecret))),
			err:      "failed to find breakglass justification token",
		},
		{
			name:     "valid",
			tokenStr: testSignToken(t, breakglassToken, jwt.WithKey(jwa.HS256, []byte(BreakglassHMACSecret))),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseBreakglassToken(tc.tokenStr)
			if diff := testutil.DiffErrString(err, tc.err); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
		})
	}
}

func testSignToken(tb testing.TB, token jwt.Token, opts ...jwt.SignOption) string {
	tb.Helper()

	b, err := jwt.Sign(token, opts...)
	if err != nil {
		tb.Fatal(err)
	}
	return string(b)
}
