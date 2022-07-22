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
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
)

// RemoteConfig for remote support of reading/writing config
type RemoteConfig interface {
	// Unmarshal read remote config and store the result in the value pointed to by 'data'
	Unmarshal(ctx context.Context, data any) error

	// Get get remote config by 'key'
	Get(ctx context.Context, key string) (any, error)

	// Set set remote config by 'key', accepts simpler form of field path as a string in which the individual fields are separated by dots as the key.
	Set(ctx context.Context, key string, value any) error
}

// FirestoreConfig for support of reading/writing config in Firestore
type FirestoreConfig struct {
	client      *firestore.Client
	docFullPath string
}

// NewFirestoreConfig allocates and returns a new FirestoreConfig which is used to reading/writing config stored at location pointed by `docFullPath`.
func NewFirestoreConfig(client *firestore.Client, docFullPath string) *FirestoreConfig {
	return &FirestoreConfig{
		client:      client,
		docFullPath: docFullPath,
	}
}

func (fireStoreRemoteCfg *FirestoreConfig) Unmarshal(ctx context.Context, data any) error {
	snap, err := fireStoreRemoteCfg.client.Doc(fireStoreRemoteCfg.docFullPath).Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to read from FireStore Doc %s: %w", fireStoreRemoteCfg.docFullPath, err)
	}
	if err = snap.DataTo(data); err != nil {
		return fmt.Errorf("failed to use firestore document's fields to populate struct: %w", err)
	}
	return nil
}

func (fireStoreRemoteCfg *FirestoreConfig) Get(ctx context.Context, key string) (any, error) {
	snap, err := fireStoreRemoteCfg.client.Doc(fireStoreRemoteCfg.docFullPath).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read from FireStore Doc %s: %w", fireStoreRemoteCfg.docFullPath, err)
	}
	value, err := snap.DataAt(key)
	if err != nil {
		return nil, fmt.Errorf("failed to read from FireStore Doc %s Key %s: %w", fireStoreRemoteCfg.docFullPath, key, err)
	}
	return value, nil
}

func (fireStoreRemoteCfg *FirestoreConfig) Set(ctx context.Context, key string, value any) error {
	doc := fireStoreRemoteCfg.client.Doc(fireStoreRemoteCfg.docFullPath)
	if _, err := doc.Update(ctx, []firestore.Update{{Path: key, Value: value}}); err != nil {
		return fmt.Errorf("failed to update remote config with key %s: %w", key, err)
	}
	return nil
}
