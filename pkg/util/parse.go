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

// Package util provides utilities that are intended to enable easier
// and more concise writing of source code.
package util

import (
	"fmt"
	"strings"
)

// FirestoreDoc represents a Firestore document, used to set up FirestoreConfig.
type FirestoreDoc struct {
	ProjectID string
	DocPath   string
}

// ParseFirestoreDocResource construct FirestoreDoc struct based on resource path of Firestore document.
func ParseFirestoreDocResource(resourceName string) (*FirestoreDoc, error) {
	parsedStrs := strings.SplitN(resourceName, "/", 6)
	if parsedStrs[0] != "projects" || parsedStrs[2] != "databases" || parsedStrs[4] != "documents" || len(parsedStrs) != 6 {
		return nil, fmt.Errorf("failed to parse firestore doc resource %v", resourceName)
	}
	return &FirestoreDoc{
		ProjectID: parsedStrs[1],
		DocPath:   parsedStrs[len(parsedStrs)-1],
	}, nil
}
