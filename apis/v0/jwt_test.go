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

package v0

import (
	"testing"
	"time"
	"unsafe"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"

	"github.com/abcxyz/pkg/testutil"
)

func TestGetRequestor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		token  jwt.Token
		exp    string
		expErr string
	}{
		{
			name:   "nil_token",
			token:  nil,
			expErr: "token cannot be nil",
		},
		{
			name:  "no_req",
			token: testTokenBuilder(t, jwt.NewBuilder()),
			exp:   "",
		},
		{
			name: "wrong_type",
			token: testTokenBuilder(t, jwt.
				NewBuilder().
				Claim(RequestorKey, []string{"not_valid"})),
			expErr: "unknown type",
		},
		{
			name: "returns_requestor",
			token: testTokenBuilder(t, jwt.
				NewBuilder().
				Claim(RequestorKey, "user@example.com"),
			),
			exp: "user@example.com",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			justs, err := GetRequestor(tc.token)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tc.exp, justs, cmpopts.IgnoreUnexported(Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestSetRequestor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		token  jwt.Token
		exp    string
		expErr string
	}{
		{
			name:   "nil_token",
			token:  nil,
			expErr: "token cannot be nil",
		},
		{
			name:  "sets",
			token: testTokenBuilder(t, jwt.NewBuilder()),
			exp:   "user@example.com",
		},
		{
			name: "overwrites",
			token: testTokenBuilder(t, jwt.
				NewBuilder().
				Claim(RequestorKey, "foo@bar.com"),
			),
			exp: "user@example.com",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := SetRequestor(tc.token, "user@example.com")
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if err != nil {
				return
			}

			got, err := GetRequestor(tc.token)
			if err != nil {
				t.Fatal(err)
			}

			if got, want := got, tc.exp; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}

func TestClearRequestor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		token  jwt.Token
		exp    string
		expErr string
	}{
		{
			name:   "nil_token",
			token:  nil,
			expErr: "token cannot be nil",
		},
		{
			name:  "clears_empty",
			token: testTokenBuilder(t, jwt.NewBuilder()),
			exp:   "",
		},
		{
			name: "clears_set",
			token: testTokenBuilder(t, jwt.
				NewBuilder().
				Claim(RequestorKey, "user@example.com"),
			),
			exp: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ClearRequestor(tc.token)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if err != nil {
				return
			}

			justs, err := GetRequestor(tc.token)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.exp, justs, cmpopts.IgnoreUnexported(Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestGetJustifications(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	cases := []struct {
		name   string
		token  jwt.Token
		exp    []*Justification
		expErr string
	}{
		{
			name:   "nil_token",
			token:  nil,
			expErr: "token cannot be nil",
		},
		{
			name:  "no_justs",
			token: testTokenBuilder(t, jwt.NewBuilder()),
			exp:   []*Justification{},
		},
		{
			name: "wrong_type",
			token: testTokenBuilder(t, jwt.
				NewBuilder().
				Claim(JustificationsKey, "not_valid")),
			expErr: "unknown type",
		},
		{
			// This test checks that we still properly decode justifications even if
			// the caller did not specify decoding the custom type claims. To drop all
			// type information, we serialize the token and then parse it without type
			// information.
			name: "not_decoded_claims",
			token: func() jwt.Token {
				token, err := jwt.NewBuilder().
					Claim(JustificationsKey, []*Justification{
						{
							Category: "category",
							Value:    "value",
						},
					}).
					Build()
				if err != nil {
					t.Fatal(err)
				}

				b, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, []byte("KEY")))
				if err != nil {
					t.Fatal(err)
				}

				parsed, err := jwt.ParseInsecure(b,
					jwt.WithContext(ctx),
					jwt.WithAcceptableSkew(5*time.Second),
					// WithTypedJustifications(), // explicitly do not use typed claims
				)
				if err != nil {
					t.Fatal(err)
				}
				return parsed
			}(),
			exp: []*Justification{
				{
					Category: "category",
					Value:    "value",
				},
			},
		},
		{
			name: "single_justification",
			token: testTokenBuilder(t, jwt.
				NewBuilder().
				Claim(JustificationsKey, &Justification{
					Category: "category",
					Value:    "value",
				}),
			),
			exp: []*Justification{
				{
					Category: "category",
					Value:    "value",
				},
			},
		},
		{
			name: "returns_justifications",
			token: testTokenBuilder(t, jwt.
				NewBuilder().
				Claim(JustificationsKey, []*Justification{
					{
						Category: "category",
						Value:    "value",
					},
				}),
			),
			exp: []*Justification{
				{
					Category: "category",
					Value:    "value",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			justs, err := GetJustifications(tc.token)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tc.exp, justs, cmpopts.IgnoreUnexported(Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestSetJustifications(t *testing.T) {
	t.Parallel()

	justs := []*Justification{
		{
			Category: "category",
			Value:    "value",
		},
	}

	cases := []struct {
		name   string
		token  jwt.Token
		exp    []*Justification
		expErr string
	}{
		{
			name:   "nil_token",
			token:  nil,
			expErr: "token cannot be nil",
		},
		{
			name:  "sets",
			token: testTokenBuilder(t, jwt.NewBuilder()),
			exp:   justs,
		},
		{
			name: "overwrites",
			token: testTokenBuilder(t, jwt.
				NewBuilder().
				Claim(JustificationsKey, []*Justification{
					{
						Category: "old",
						Value:    "value",
					},
				}),
			),
			exp: justs,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := SetJustifications(tc.token, justs)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if err != nil {
				return
			}

			if diff := cmp.Diff(tc.exp, justs, cmpopts.IgnoreUnexported(Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}

			if unsafe.Pointer(&tc.exp) == unsafe.Pointer(&justs) {
				t.Error("expected result to be a copy")
			}
		})
	}
}

func TestClearJustifications(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		token  jwt.Token
		exp    []*Justification
		expErr string
	}{
		{
			name:   "nil_token",
			token:  nil,
			expErr: "token cannot be nil",
		},
		{
			name:  "sets",
			token: testTokenBuilder(t, jwt.NewBuilder()),
			exp:   []*Justification{},
		},
		{
			name: "overwrites",
			token: testTokenBuilder(t, jwt.
				NewBuilder().
				Claim(JustificationsKey, []*Justification{
					{
						Category: "category",
						Value:    "value",
					},
				}),
			),
			exp: []*Justification{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := ClearJustifications(tc.token)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if err != nil {
				return
			}

			justs, err := GetJustifications(tc.token)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.exp, justs, cmpopts.IgnoreUnexported(Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func testTokenBuilder(tb testing.TB, b *jwt.Builder) jwt.Token {
	tb.Helper()

	if b == nil {
		return nil
	}

	token, err := b.Build()
	if err != nil {
		tb.Fatal(err)
	}
	return token
}
