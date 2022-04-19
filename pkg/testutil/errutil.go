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

// Package testutil provides utilities that are intended to enable easier
// and more concise writing of unit test code.
package testutil

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// ErrCmp compares an expected error string with a received error for use in testing.
func ErrCmp(t *testing.T, wantErr string, gotErr error) {
	if wantErr != "" {
		if gotErr != nil {
			if diff := cmp.Diff(gotErr.Error(), wantErr); diff != "" {
				t.Errorf("Process got unexpected error substring: %v", diff)
			}
		} else {
			t.Errorf("Expected error, but received nil")
		}
	} else if gotErr != nil {
		t.Errorf("Expected no error, but received \"%v\"", gotErr)
	}
}
