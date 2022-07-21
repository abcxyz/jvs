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
	LoadRemoteConfigTo(ctx context.Context, data interface{}) error

	UpdateRemoteConfig(ctx context.Context, key string, value interface{}) error // configPath represent the fields that reference a value, accepts simpler form of field path as a string in which the individual fields are separated by '/' as the configPath.
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

func (fireStoreRemoteCfg FirestoreRemoteConfig) LoadRemoteConfigTo(ctx context.Context, data interface{}) error {
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

// UpdateRemoteConfig :Firestore implementation accepts simpler form of field path as a string in which the individual fields are separated by dots as the configPath.
func (fireStoreRemoteCfg FirestoreRemoteConfig) UpdateRemoteConfig(ctx context.Context, key string, value interface{}) error {
	doc := fireStoreRemoteCfg.client.Doc(fireStoreRemoteCfg.docFullPath)
	if key == "" {
		if _, err := doc.Set(ctx, value); err != nil {
			return fmt.Errorf("failed to set remote config: %w", err)
		}
	} else {
		_, err := doc.Update(ctx, []firestore.Update{{Path: key, Value: value}})
		if err != nil {
			return fmt.Errorf("failed to update remote config with key %s: %w", key, err)
		}
	}
	return nil
}
