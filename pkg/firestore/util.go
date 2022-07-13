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

package firestore

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

// [START firestore_data_custom_type_definition]

const (
	Collection             = "JVS"
	JustificationConfigDoc = "JustificationConfig"
	PublicKeyConfigDoc     = "PublicKeyConfig"
	CertRotationConfigDoc  = "CertRotationConfig"
)

type KMSConfig struct {
	// KeyNames format: `[projects/*/locations/*/keyRings/*/cryptoKeys/*]`
	KeyNames []string `firestore:"key_names,omitempty"`
}

// [END firestore_data_custom_type_definition]

func GetKMSConfig(ctx context.Context, fsClient *firestore.Client, collectionPath, docID string) (*KMSConfig, error) {
	kSnap, err := fsClient.Collection(collectionPath).Doc(docID).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read from Collection %s Doc %s: %w", collectionPath, docID, err)
	}
	var kmsConfig KMSConfig
	err = kSnap.DataTo(&kmsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to use firestore document's fields to populate struct: %w", err)
	}
	return &kmsConfig, nil
}
