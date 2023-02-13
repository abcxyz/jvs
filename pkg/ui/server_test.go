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
	"fmt"
	"testing"

	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

type testValidateFormParam struct {
	name   string
	detail FormDetails
	want   bool
}

func TestValidateOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		origin    string
		allowList []string
		wantRes   bool
		wantErr   string
	}{
		{
			name:      "no_origin_and_empty_allow_list",
			origin:    "",
			allowList: []string{},
			wantRes:   false,
		},
		{
			name:      "no_origin",
			origin:    "",
			allowList: []string{"foo.com"},
			wantRes:   false,
		},
		{
			name:      "origin_domain_no_match",
			origin:    "bar.com",
			allowList: []string{"foo.com"},
			wantRes:   false,
		},
		{
			name:      "origin_subdomain_no_match",
			origin:    "go.foo.com",
			allowList: []string{"bar.com", "baz.com"},
			wantRes:   false,
		},
		{
			name:      "origin_match_asterisk",
			origin:    "foo.com",
			allowList: []string{"*"},
			wantRes:   true,
		},
		{
			name:      "subdomain_origin_match",
			origin:    "go.foo.com",
			allowList: []string{"bar.com", "foo.com"},
			wantRes:   true,
		},
		{
			name:      "domain_origin_match",
			origin:    "foo.com",
			allowList: []string{"bar.com", "foo.com"},
			wantRes:   true,
		},
		{
			name:      "local_origin",
			origin:    "localhost",
			allowList: []string{"*"},
			wantRes:   true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotRes, err := validateOrigin(tc.origin, tc.allowList)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if diff := cmp.Diff(tc.wantRes, gotRes); diff != "" {
				t.Errorf("Failed validating (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestValidateLocalIp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{
			name:   "no_origin",
			origin: "",
			want:   false,
		},
		{
			name:   "missing_protocol",
			origin: "localhost",
			want:   false,
		},
		{
			name:   "localhost_origin",
			origin: "http://localhost",
			want:   true,
		},
		{
			name:   "local_ip_origin",
			origin: "http://127.0.0.1",
			want:   true,
		},
		{
			name:   "non_local_ip_origin",
			origin: "google.com",
			want:   false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, _ := validateLocalIP(tc.origin)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Failed validating (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestValidateForm(t *testing.T) {
	t.Parallel()

	var tests []testValidateFormParam

	for i := 0; i < len(categories); i++ {
		category := categories[i]
		for j := 0; j < len(ttls); j++ {
			ttl := ttls[j]
			reason := "reason"
			happyPathCase := testValidateFormParam{
				name: fmt.Sprintf("%s_%s_%s", category, reason, ttl),
				detail: FormDetails{
					Category: category,
					Reason:   reason,
					TTL:      ttl,
				},
				want: true,
			}
			tests = append(tests, happyPathCase)
		}
	}

	sadPathCases := []testValidateFormParam{
		{
			name: "empty_input_all",
			detail: FormDetails{
				Category: "",
				Reason:   "",
				TTL:      "",
			},
			want: false,
		},
		{
			name: "empty_input_category",
			detail: FormDetails{
				Category: "",
				Reason:   "reason",
				TTL:      ttls[0],
			},
			want: false,
		},
		{
			name: "empty_input_reason",
			detail: FormDetails{
				Category: categories[0],
				Reason:   "",
				TTL:      ttls[1],
			},
			want: false,
		},
		{
			name: "empty_input_ttl",
			detail: FormDetails{
				Category: categories[0],
				Reason:   "reason",
				TTL:      "",
			},
			want: false,
		},
	}

	tests = append(tests, sadPathCases...)

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := validateForm(&tc.detail)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Failed validating (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestIsValidOneOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		selection string
		options   []string
		want      bool
	}{
		{
			name:      "empty_selection_and_options_input",
			selection: "",
			options:   []string{},
			want:      false,
		},
		{
			name:      "empty_options_input",
			selection: "foo",
			options:   []string{},
			want:      false,
		},
		{
			name:      "selection_not_in_options",
			selection: "foo",
			options:   []string{"bar"},
			want:      false,
		},
		{
			name:      "selection_in_options",
			selection: "foo",
			options:   []string{"bar", "foo"},
			want:      true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isValidOneOf(tc.selection, tc.options)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Failed validating (-want,+got):\n%s", diff)
			}
		})
	}
}
