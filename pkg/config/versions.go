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

import "sort"

// VersionList is a set of allowed versions. Create with NewVersionList.
type VersionList struct {
	m map[string]struct{}
}

// NewVersionList creates an efficient list of allowed version strings and
// exposes functions for efficiently querying membership.
func NewVersionList(versions ...string) *VersionList {
	m := make(map[string]struct{}, len(versions))
	for _, v := range versions {
		m[v] = struct{}{}
	}

	return &VersionList{
		m: m,
	}
}

// Contains returns true if the given version string is an allowed version in
// the list, or false otherwise.
func (vl *VersionList) Contains(version string) bool {
	if vl == nil || vl.m == nil {
		return false
	}

	_, ok := vl.m[version]
	return ok
}

// List returns a copy of the list of allowed versions, usually for displaying
// in an error message.
func (vl *VersionList) List() []string {
	if vl == nil || vl.m == nil {
		return []string{}
	}

	l := make([]string, 0, len(vl.m))
	for key := range vl.m {
		l = append(l, key)
	}
	sort.Strings(l)
	return l
}
