package jvscrypto

import (
	"context"
	"fmt"
	"strings"

	kms "cloud.google.com/go/kms/apiv1"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type StateStore interface {
	// WriteVersionState writes a state for a version to the state store.
	WriteVersionState(ctx context.Context, key string, versionName string, state VersionState) error

	// RemoveVersion removes a version from the state store.
	RemoveVersion(ctx context.Context, key string, versionName string) error

	// GetActiveVersionStates returns a map from version name (full KMS name) to VersionState for active versions.
	GetActiveVersionStates(ctx context.Context, key string) (map[string]VersionState, error)
}

// VersionState is independent of KMS states. This allows us to distinguish which key (regardless of KMS state)
// to use when signing. These are stored in the key labels.
type VersionState int64

const (
	VersionStatePrimary VersionState = iota
	VersionStateNew
	VersionStateOld
	VersionStateUnknown
)

func (v VersionState) String() string {
	switch v {
	case VersionStatePrimary:
		return "primary"
	case VersionStateNew:
		return "new"
	case VersionStateOld:
		return "old"
	}
	return "unknown"
}

// GetVersionState converts a string to a VersionState.
func GetVersionState(s string) VersionState {
	switch s {
	case "primary":
		return VersionStatePrimary
	case "new":
		return VersionStateNew
	case "old":
		return VersionStateOld
	}
	return VersionStateUnknown
}

type KeyLabelStateStore struct {
	KmsClient *kms.KeyManagementClient
}

func (k *KeyLabelStateStore) WriteVersionState(ctx context.Context, key string, versionName string, state VersionState) error {
	response, err := k.KmsClient.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{Name: key})
	if err != nil {
		return fmt.Errorf("issue while getting key from KMS: %w", err)
	}

	verName, err := getLabelKey(versionName)
	if err != nil {
		return err
	}
	// update label
	labels := response.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[verName] = state.String()
	response.Labels = labels

	var messageType *kmspb.CryptoKey
	mask, err := fieldmaskpb.New(messageType, "labels")
	if err != nil {
		return err
	}
	_, err = k.KmsClient.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{CryptoKey: response, UpdateMask: mask})
	if err != nil {
		return fmt.Errorf("issue while setting labels in kms %w", err)
	}
	return nil
}

func (k *KeyLabelStateStore) RemoveVersion(ctx context.Context, key string, versionName string) error {
	response, err := k.KmsClient.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{Name: key})
	if err != nil {
		return fmt.Errorf("issue while getting key from KMS: %w", err)
	}

	verName, err := getLabelKey(versionName)
	if err != nil {
		return err
	}

	// delete label
	labels := response.Labels
	delete(labels, verName)
	response.Labels = labels

	var messageType *kmspb.CryptoKey
	mask, err := fieldmaskpb.New(messageType, "labels")
	if err != nil {
		return err
	}
	k.KmsClient.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{CryptoKey: response, UpdateMask: mask})
	return nil
}

func (k *KeyLabelStateStore) GetActiveVersionStates(ctx context.Context, key string) (map[string]VersionState, error) {
	response, err := k.KmsClient.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{Name: key})
	if err != nil {
		return nil, fmt.Errorf("issue while getting key from KMS: %w", err)
	}
	vers := make(map[string]VersionState)
	for ver, state := range response.Labels {
		ver = strings.TrimPrefix(ver, "ver_")
		verNameWithPrefix := fmt.Sprintf("%s/cryptoKeyVersions/%s", key, ver)
		vers[verNameWithPrefix] = GetVersionState(state)
	}
	return vers, nil
}

// This returns the key version name with "ver_" prefixed. This is because labels must start with a lowercase letter, and can't go over 64 chars.
func getLabelKey(versionName string) (string, error) {
	split := strings.Split(versionName, "/")
	if len(split) != 10 {
		return "", fmt.Errorf("input had unexpected format: \"%s\"", versionName)
	}
	versionWithoutPrefix := "ver_" + split[len(split)-1]
	return versionWithoutPrefix, nil
}
