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

// Package util_test includes the Unit tests of package util
package util_test

import (
	"testing"

	"github.com/abcxyz/jvs/pkg/util"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

func TestParseFirestoreDocResource(t *testing.T) {
	tests := []struct {
		name                 string
		firestoreDocResource string
		wantFirestoreDoc     *util.FirestoreDoc
		wantErrStr           string
	}{
		{
			name:                 "valid_resource",
			firestoreDocResource: "projects/test-project/databases/(default)/documents/jvs/key_config",
			wantFirestoreDoc: &util.FirestoreDoc{
				ProjectID: "test-project",
				DocPath:   "jvs/key_config",
			},
		},
		{
			name:                 "invalid_resource",
			firestoreDocResource: "projects/test-project/documents/jvs/key_config",
			wantErrStr:           "failed to parse firestore doc resource",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := util.ParseFirestoreDocResource(tc.firestoreDocResource)
			if diff := testutil.DiffErrString(err, tc.wantErrStr); diff != "" {
				t.Errorf("ParseFirestoreDocResource got unexpected error substring: %v", diff)
			}

			if diff := cmp.Diff(tc.wantFirestoreDoc, got); diff != "" {
				t.Errorf("FirestoreDoc diff (-want, +got): %v", diff)
			}
		})
	}
}
