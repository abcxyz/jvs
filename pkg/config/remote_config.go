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

type RemoteConfig interface {
	Load(ctx context.Context, data any) error

	GetByKey(ctx context.Context, key string) (any, error)

	// SetByKey accepts simpler form of field path as a string in which the individual fields are separated by dots as the key.
	SetByKey(ctx context.Context, key string, value any) error
}

type FirestoreRemoteConfig struct {
	client      *firestore.Client
	docFullPath string
}

func NewFirestoreRemoteConfig(client *firestore.Client, docFullPath string) FirestoreRemoteConfig {
	return FirestoreRemoteConfig{
		client:      client,
		docFullPath: docFullPath,
	}
}

func (fireStoreRemoteCfg FirestoreRemoteConfig) Load(ctx context.Context, data any) error {
	snap, err := fireStoreRemoteCfg.client.Doc(fireStoreRemoteCfg.docFullPath).Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to read from FireStore Doc %s: %w", fireStoreRemoteCfg.docFullPath, err)
	}
	err = snap.DataTo(data)
	if err != nil {
		return fmt.Errorf("failed to use firestore document's fields to populate struct: %w", err)
	}
	return nil
}

func (fireStoreRemoteCfg FirestoreRemoteConfig) GetByKey(ctx context.Context, key string) (any, error) {
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

func (fireStoreRemoteCfg FirestoreRemoteConfig) SetByKey(ctx context.Context, key string, value any) error {
	doc := fireStoreRemoteCfg.client.Doc(fireStoreRemoteCfg.docFullPath)
	if _, err := doc.Update(ctx, []firestore.Update{{Path: key, Value: value}}); err != nil {
		return fmt.Errorf("failed to update remote config with key %s: %w", key, err)
	}
	return nil
}
