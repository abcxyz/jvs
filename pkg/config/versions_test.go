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

package config

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewVersionList(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input []string
		exp   *VersionList
	}{
		{
			name:  "nil",
			input: nil,
			exp: &VersionList{
				m: map[string]struct{}{},
			},
		},
		{
			name:  "empty",
			input: []string{""},
			exp: &VersionList{
				m: map[string]struct{}{
					"": {},
				},
			},
		},
		{
			name:  "single",
			input: []string{"v1alpha"},
			exp: &VersionList{
				m: map[string]struct{}{
					"v1alpha": {},
				},
			},
		},
		{
			name:  "multiple",
			input: []string{"v1alpha", "v1"},
			exp: &VersionList{
				m: map[string]struct{}{
					"v1alpha": {},
					"v1":      {},
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := NewVersionList(tc.input...)
			if diff := cmp.Diff(got, tc.exp, cmp.AllowUnexported(VersionList{})); diff != "" {
				t.Errorf("Config unexpected diff (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestVersionList_Contains(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		versionList *VersionList
		input       string
		exp         bool
	}{
		{
			name:        "nil",
			versionList: nil,
			input:       "v1",
			exp:         false,
		},
		{
			name: "nil_map",
			versionList: &VersionList{
				m: nil,
			},
			input: "v1",
			exp:   false,
		},
		{
			name: "exists",
			versionList: &VersionList{
				m: map[string]struct{}{
					"v1alpha": {},
					"v1":      {},
				},
			},
			input: "v1",
			exp:   true,
		},
		{
			name: "not_exists",
			versionList: &VersionList{
				m: map[string]struct{}{
					"v1alpha": {},
					"v1":      {},
				},
			},
			input: "v2",
			exp:   false,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got, want := tc.versionList.Contains(tc.input), tc.exp; got != want {
				t.Errorf("expected %t to be %t", got, want)
			}
		})
	}
}

func TestVersionList_List(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		versionList *VersionList
		exp         []string
	}{
		{
			name:        "nil",
			versionList: nil,
			exp:         []string{},
		},
		{
			name: "nil_map",
			versionList: &VersionList{
				m: nil,
			},
			exp: []string{},
		},
		{
			name: "sorts",
			versionList: &VersionList{
				m: map[string]struct{}{
					"v1alpha": {},
					"v1":      {},
					"v2":      {},
					"_v1":     {},
				},
			},
			exp: []string{
				"_v1",
				"v1",
				"v1alpha",
				"v2",
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.versionList.List()
			if diff := cmp.Diff(got, tc.exp); diff != "" {
				t.Errorf("Config unexpected diff (-want, +got):\n%s", diff)
			}
		})
	}
}
