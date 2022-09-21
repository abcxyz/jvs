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
	"unsafe"

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestGetJustifications(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		tokenFunc func(tb testing.TB) jwt.Token
		exp       []*Justification
		expErr    string
	}{
		{
			name: "nil_token",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				return nil
			},
			expErr: "token cannot be nil",
		},
		{
			name: "no_justs",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				token, err := jwt.NewBuilder().Build()
				if err != nil {
					tb.Fatal(err)
				}
				return token
			},
			exp: []*Justification{},
		},
		{
			name: "wrong_type",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				token, err := jwt.NewBuilder().
					Claim(jwtJustificationsKey, "not_valid").
					Build()
				if err != nil {
					tb.Fatal(err)
				}
				return token
			},
			expErr: "found justifications, but was string",
		},
		{
			name: "returns_justifications",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				token, err := jwt.NewBuilder().
					Claim(jwtJustificationsKey, []*Justification{
						{
							Category: "category",
							Value:    "value",
						},
					}).
					Build()
				if err != nil {
					tb.Fatal(err)
				}
				return token
			},
			exp: []*Justification{
				{
					Category: "category",
					Value:    "value",
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			token := tc.tokenFunc(t)
			justs, err := GetJustifications(token)
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
		name      string
		tokenFunc func(tb testing.TB) jwt.Token
		exp       []*Justification
		expErr    string
	}{
		{
			name: "nil_token",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				return nil
			},
			expErr: "token cannot be nil",
		},
		{
			name: "sets",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				token, err := jwt.NewBuilder().Build()
				if err != nil {
					tb.Fatal(err)
				}
				return token
			},
			exp: justs,
		},
		{
			name: "overwrites",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				token, err := jwt.NewBuilder().
					Claim(jwtJustificationsKey, []*Justification{
						{
							Category: "category",
							Value:    "value",
						},
					}).
					Build()
				if err != nil {
					tb.Fatal(err)
				}
				return token
			},
			exp: justs,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			token := tc.tokenFunc(t)
			err := SetJustifications(token, justs)
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

func TestAppendJustification(t *testing.T) {
	t.Parallel()

	just := &Justification{
		Category: "category2",
		Value:    "value2",
	}

	cases := []struct {
		name      string
		tokenFunc func(tb testing.TB) jwt.Token
		exp       []*Justification
		expErr    string
	}{
		{
			name: "nil_token",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				return nil
			},
			expErr: "token cannot be nil",
		},
		{
			name: "appends_empty",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				token, err := jwt.NewBuilder().Build()
				if err != nil {
					tb.Fatal(err)
				}
				return token
			},
			exp: []*Justification{
				{
					Category: "category2",
					Value:    "value2",
				},
			},
		},
		{
			name: "appends_existing",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				token, err := jwt.NewBuilder().
					Claim(jwtJustificationsKey, []*Justification{
						{
							Category: "category",
							Value:    "value",
						},
					}).
					Build()
				if err != nil {
					tb.Fatal(err)
				}
				return token
			},
			exp: []*Justification{
				{
					Category: "category",
					Value:    "value",
				},
				{
					Category: "category2",
					Value:    "value2",
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			token := tc.tokenFunc(t)
			err := AppendJustification(token, just)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if err != nil {
				return
			}

			justs, err := GetJustifications(token)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.exp, justs, cmpopts.IgnoreUnexported(Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestClearJustifications(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		tokenFunc func(tb testing.TB) jwt.Token
		exp       []*Justification
		expErr    string
	}{
		{
			name: "nil_token",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				return nil
			},
			expErr: "token cannot be nil",
		},
		{
			name: "sets",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				token, err := jwt.NewBuilder().Build()
				if err != nil {
					tb.Fatal(err)
				}
				return token
			},
			exp: []*Justification{},
		},
		{
			name: "overwrites",
			tokenFunc: func(tb testing.TB) jwt.Token {
				tb.Helper()

				token, err := jwt.NewBuilder().
					Claim(jwtJustificationsKey, []*Justification{
						{
							Category: "category",
							Value:    "value",
						},
					}).
					Build()
				if err != nil {
					tb.Fatal(err)
				}
				return token
			},
			exp: []*Justification{},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			token := tc.tokenFunc(t)
			err := ClearJustifications(token)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if err != nil {
				return
			}

			justs, err := GetJustifications(token)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(tc.exp, justs, cmpopts.IgnoreUnexported(Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}
		})
	}
}
