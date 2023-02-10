// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"testing"
)

type testValidateOriginParam struct {
	origin    string
	allowList []string
	expected  bool
}

type testValidateFormParam struct {
	detail   FormDetails
	expected bool
}

type testIsValidOneOfInputParam struct {
	selection string
	options   []string
	expected  bool
}

func TestValidateOrigin(t *testing.T) {
	t.Parallel()

	inputs := []testValidateOriginParam{
		{
			origin:    "",
			allowList: []string{},
			expected:  false,
		},
		{
			origin:    "",
			allowList: []string{"foo.com"},
			expected:  false,
		},
		{
			origin:    "bar.com",
			allowList: []string{"foo.com"},
			expected:  false,
		},
		{
			origin:    "go.foo.com",
			allowList: []string{"bar.com", "baz.com"},
			expected:  false,
		},
		{
			origin:    "go.foo.com",
			allowList: []string{"bar.com", "foo.com"},
			expected:  true,
		},
		{
			origin:    "foo.com",
			allowList: []string{"bar.com", "foo.com"},
			expected:  true,
		},
		{
			origin:    "foo.com",
			allowList: []string{"bar.com", "*"},
			expected:  true,
		},
	}

	for _, value := range inputs {
		if validateOrigin(value.origin, value.allowList) != value.expected {
			t.Fatalf("Failed validating %+v", value)
		}
	}
}

func TestValidateForm(t *testing.T) {
	t.Parallel()

	var inputs []testValidateFormParam

	for i := 0; i < len(categories); i++ {
		category := categories[i]
		for j := 0; j < len(ttls); j++ {
			ttl := ttls[j]

			happyPathCase := testValidateFormParam{
				detail: FormDetails{
					Category: category,
					Reason:   "reason",
					TTL:      ttl,
				},
				expected: true,
			}

			inputs = append(inputs, happyPathCase)
		}
	}

	sadPathCases := []testValidateFormParam{
		{
			detail: FormDetails{
				Category: "",
				Reason:   "",
				TTL:      "",
			},
			expected: false,
		},
		{
			detail: FormDetails{
				Category: "",
				Reason:   "reason",
				TTL:      ttls[0],
			},
			expected: false,
		},
		{
			detail: FormDetails{
				Category: categories[0],
				Reason:   "",
				TTL:      ttls[1],
			},
			expected: false,
		},
		{
			detail: FormDetails{
				Category: categories[0],
				Reason:   "reason",
				TTL:      "",
			},
			expected: false,
		},
	}

	inputs = append(inputs, sadPathCases...)

	for _, value := range inputs {
		if validateForm(&value.detail) != value.expected {
			t.Fatalf("Failed validating form %+v", value)
		}
	}
}

func TestIsValidOneOf(t *testing.T) {
	t.Parallel()

	inputs := []testIsValidOneOfInputParam{
		{
			selection: "",
			options:   []string{},
			expected:  false,
		},
		{
			selection: "foo",
			options:   []string{},
			expected:  false,
		},
		{
			selection: "foo",
			options:   []string{"bar"},
			expected:  false,
		},
		{
			selection: "foo",
			options:   []string{"bar", "foo"},
			expected:  true,
		},
	}

	for _, value := range inputs {
		if isValidOneOf(value.selection, value.options) != value.expected {
			t.Fatalf("Failed validating %+v", value)
		}
	}
}
