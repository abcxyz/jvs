package config

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
)

type RemoteConfig interface {
	GetRemoteConfigTo(ctx context.Context, data interface{}) error

	UpdateRemoteConfig(ctx context.Context, configPath string, value interface{}) error // configPath represent the fields that reference a value, accepts simpler form of field path as a string in which the individual fields are separated by '/' as the configPath
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

func (fireStoreRemoteCfg FirestoreRemoteConfig) GetRemoteConfigTo(ctx context.Context, data interface{}) error {
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

// UpdateRemoteConfig :Firestore implementation accepts simpler form of field path as a string in which the individual fields are separated by dots as the configPath
func (fireStoreRemoteCfg FirestoreRemoteConfig) UpdateRemoteConfig(ctx context.Context, configPath string, value interface{}) error {
	doc := fireStoreRemoteCfg.client.Doc(fireStoreRemoteCfg.docFullPath)
	s := strings.Split(configPath, "/")
	path := strings.Join(s[:], ".")
	_, err := doc.Update(ctx, []firestore.Update{{Path: path, Value: value}})
	if err != nil {
		return err
	}
	return nil
}
