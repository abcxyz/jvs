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

package jvscrypto

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/sethvargo/go-retry"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/pkg/logging"
)

// RotationHandler handles all necessary rotation actions for asymmetric keys
// based off a provided configuration.
type RotationHandler struct {
	kmsClient *kms.KeyManagementClient
	config    *config.CertRotationConfig
}

// NewRotationHandler creates a handler for rotating keys.
func NewRotationHandler(ctx context.Context, kmsClient *kms.KeyManagementClient, cfg *config.CertRotationConfig) *RotationHandler {
	if cfg == nil {
		cfg = &config.CertRotationConfig{}
	}

	return &RotationHandler{
		kmsClient: kmsClient,
		config:    cfg,
	}
}

// RotateKeys rotates all keys.
func (h *RotationHandler) RotateKeys(ctx context.Context) (merr error) {
	logger := logging.FromContext(ctx)

	// TODO: load keys from DB instead. https://github.com/abcxyz/jvs/issues/17
	for _, key := range h.config.KeyNames {
		if err := h.RotateKey(ctx, key); err != nil {
			merr = errors.Join(merr, fmt.Errorf("failed to rotate key %s: %w", key, err))
			continue
		}
		logger.InfoContext(ctx, "successfully rotated (if necessary)", "key", key)
	}

	return
}

// RotateKey is called to determine and perform rotation actions on versions for a key.
// key is the full resource name: `projects/*/locations/*/keyRings/*/cryptoKeys/*`
// https://pkg.go.dev/google.golang.org/genproto/googleapis/cloud/kms/v1#CryptoKey
func (h *RotationHandler) RotateKey(ctx context.Context, key string) error {
	curTime := time.Now().UTC()
	// Create the request to list Keys.
	listKeysReq := &kmspb.ListCryptoKeyVersionsRequest{
		Parent: key,
	}

	// List the Key Versions in the Key
	it := h.kmsClient.ListCryptoKeyVersions(ctx, listKeysReq)
	vers := make([]*kmspb.CryptoKeyVersion, 0)
	for {
		ver, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("err while reading crypto key version list: %w", err)
		}
		vers = append(vers, ver)
	}

	// Get any relevant Key Version information from the StateStore
	primaryName, err := GetPrimary(ctx, h.kmsClient, key)
	if err != nil {
		return fmt.Errorf("failed to determine primary: %w", err)
	}
	actions, err := h.determineActions(ctx, vers, primaryName, curTime)
	if err != nil {
		return fmt.Errorf("failed to determine cert actions: %w", err)
	}

	if err = h.performActions(ctx, key, actions); err != nil {
		return fmt.Errorf("failed to perform some cert actions: %w", err)
	}
	return nil
}

type Action int8

const (
	ActionCreateNew           Action = iota // New version should be created. Will be marked as new in StateStore (SS).
	ActionCreateNewAndPromote               // New version should be created. Will be marked as primary in SS.
	ActionPromote                           // Mark version as primary in SS.
	ActionDisable                           // Disable version. Will be removed from SS.
	ActionDestroy                           // Destroy version.
)

type actionTuple struct {
	Action  Action
	Version *kmspb.CryptoKeyVersion
}

func (h *RotationHandler) determineActions(ctx context.Context, vers []*kmspb.CryptoKeyVersion, primaryName string, curTime time.Time) ([]*actionTuple, error) {
	logger := logging.FromContext(ctx)
	var primary *kmspb.CryptoKeyVersion
	var olderVers []*kmspb.CryptoKeyVersion
	var newerVers []*kmspb.CryptoKeyVersion

	for _, ver := range vers {
		logger.DebugContext(ctx, "checking version", "version", ver)
		if primaryName != "" && ver.GetName() == primaryName {
			primary = ver
			break
		}
	}

	if primary == nil {
		// No primary found, all are newer than nothing.
		newerVers = vers
	} else {
		for _, ver := range vers {
			if ver.GetName() == primaryName {
				continue
			}
			if createdBefore(ver, primary) {
				logger.DebugContext(ctx, "version is older", "version", ver)
				olderVers = append(olderVers, ver)
			} else {
				logger.DebugContext(ctx, "version is newer", "version", ver)
				newerVers = append(newerVers, ver)
			}
		}
	}

	actions := h.actionsForOlderVersions(ctx, olderVers, curTime)

	newActions := h.actionsForNewVersions(ctx, newerVers, primary, curTime)
	actions = append(actions, newActions...)

	return actions, nil
}

func createdBefore(ver1, ver2 *kmspb.CryptoKeyVersion) bool {
	return ver1.GetCreateTime().AsTime().Before(ver2.GetCreateTime().AsTime())
}

// Determine actions for non-primary enabled versions.
func (h *RotationHandler) actionsForNewVersions(ctx context.Context, vers []*kmspb.CryptoKeyVersion, primary *kmspb.CryptoKeyVersion, curTime time.Time) []*actionTuple {
	logger := logging.FromContext(ctx)
	actions := make([]*actionTuple, 0)
	newest := newestEnabledVer(vers)

	// If newest is eligible for promotion, promote and don't do anything with the
	// current primary.
	if h.shouldPromote(ctx, primary, newest, curTime) {
		return append(actions, &actionTuple{ActionPromote, newest})
	}

	// We don't have a version eligible for promotion. If no primary currently
	// exists, we need to create a new version and promote it to primary.
	if primary == nil {
		logger.InfoContext(ctx, "no primary or new keys found, creating a new key and immediately promoting to primary")
		return append(actions, &actionTuple{ActionCreateNewAndPromote, nil})
	}

	// We don't have a new key we're promoting, see if we should create a new key.
	if h.shouldRotate(ctx, primary, newest, curTime) {
		actions = append(actions, &actionTuple{ActionCreateNew, nil})
	}
	return actions
}

func newestEnabledVer(vers []*kmspb.CryptoKeyVersion) *kmspb.CryptoKeyVersion {
	var newest *kmspb.CryptoKeyVersion
	var newestTime time.Time

	for _, ver := range vers {
		if ver.GetState() != kmspb.CryptoKeyVersion_ENABLED {
			continue
		}
		if newest == nil || ver.GetCreateTime().AsTime().After(newestTime) {
			newest = ver
			newestTime = ver.GetCreateTime().AsTime()
		}
	}
	return newest
}

// Determine actions for disabled versions.
func (h *RotationHandler) actionsForOlderVersions(ctx context.Context, vers []*kmspb.CryptoKeyVersion, curTime time.Time) []*actionTuple {
	logger := logging.FromContext(ctx)
	actions := make([]*actionTuple, 0)

	for _, ver := range vers {
		//nolint:exhaustive // TODO: handle import cases. https://github.com/abcxyz/jvs/issues/5
		switch ver.GetState() {
		case kmspb.CryptoKeyVersion_ENABLED:
			if h.shouldDisable(ctx, ver, curTime) {
				actions = append(actions, &actionTuple{ActionDisable, ver})
			}
		case kmspb.CryptoKeyVersion_DISABLED:
			if h.shouldDestroy(ctx, ver, curTime) {
				actions = append(actions, &actionTuple{ActionDestroy, ver})
			}
		default:
			logger.InfoContext(ctx, "no action needed for key version in current state",
				"version", ver,
				"state", ver.GetState())
		}
	}
	return actions
}

func (h *RotationHandler) shouldDestroy(ctx context.Context, ver *kmspb.CryptoKeyVersion, curTime time.Time) bool {
	logger := logging.FromContext(ctx)
	cutoff := curTime.Add(-h.config.DestroyAge())
	shouldDestroy := ver.GetCreateTime().AsTime().Before(cutoff)
	if shouldDestroy {
		logger.InfoContext(ctx, "version created before cutoff date, should destroy",
			"version", ver,
			"cutoff", cutoff)
	} else {
		logger.DebugContext(ctx, "version created after cutoff date, no action necessary",
			"version", ver,
			"cutoff", cutoff)
	}
	return shouldDestroy
}

func (h *RotationHandler) shouldDisable(ctx context.Context, ver *kmspb.CryptoKeyVersion, curTime time.Time) bool {
	logger := logging.FromContext(ctx)
	cutoff := curTime.Add(-h.config.KeyTTL)
	shouldDisable := ver.GetCreateTime().AsTime().Before(cutoff)
	if shouldDisable {
		logger.InfoContext(ctx, "version created before cutoff date, should disable",
			"version", ver,
			"cutoff", cutoff)
	} else {
		logger.DebugContext(ctx, "version created after cutoff date, no action necessary",
			"version", ver,
			"cutoff", cutoff)
	}
	return shouldDisable
}

// Determines whether a new key version should be created. It returns true only
// when the newest version does not exist and the primary version reaches its
// rotation age.
func (h *RotationHandler) shouldRotate(ctx context.Context, primary, newest *kmspb.CryptoKeyVersion, curTime time.Time) bool {
	logger := logging.FromContext(ctx)
	if newest != nil {
		logger.DebugContext(ctx, "new version already created, no action necessary",
			"version", newest)
		return false
	}
	cutoff := curTime.Add(-h.config.RotationAge())
	shouldRotate := primary.GetCreateTime().AsTime().Before(cutoff)
	if shouldRotate {
		logger.InfoContext(ctx, "version created before cutoff date, should rotate",
			"version", primary,
			"cutoff", cutoff)
	} else {
		logger.DebugContext(ctx, "version created after cutoff date, no action necessary",
			"version", primary,
			"cutoff", cutoff)
	}
	return shouldRotate
}

// Determines whether the newest key version should be promoted to primary. It
// returns true when the primary version does not exist or when the newest
// version crosses its propagation delay.
func (h *RotationHandler) shouldPromote(ctx context.Context, primary, newest *kmspb.CryptoKeyVersion, curTime time.Time) bool {
	logger := logging.FromContext(ctx)
	if newest == nil {
		return false
	}
	if primary == nil {
		logger.InfoContext(ctx, "primary does not exist, should promote the newest key to primary regardless of propagation delay",
			"version", newest)
		return true
	}
	cutoff := curTime.Add(-h.config.PropagationDelay)
	canPromote := newest.GetCreateTime().AsTime().Before(cutoff)
	if canPromote {
		logger.InfoContext(ctx, "version created before cutoff date, should promote to primary",
			"version", newest,
			"cutoff", cutoff)
	} else {
		logger.DebugContext(ctx, "version created after cutoff date, no action necessary",
			"version", newest,
			"cutoff", cutoff)
	}
	return canPromote
}

// TODO: it may be worth adding rollback functionality for cases where multiple
// actions are expected to occur. for example, if we are demoting a key, we also
// need to ensure we've marked another as primary, and may end up in an odd
// state if one action occurs and the other does not.
func (h *RotationHandler) performActions(ctx context.Context, keyName string, actions []*actionTuple) (merr error) {
	logger := logging.FromContext(ctx)
	for _, action := range actions {
		switch action.Action {
		case ActionCreateNew:
			_, err := h.performCreateNew(ctx, keyName)
			if err != nil {
				merr = errors.Join(merr, err)
			}
		case ActionPromote:
			if err := SetPrimary(ctx, h.kmsClient, keyName, action.Version.GetName()); err != nil {
				merr = errors.Join(merr, err)
			}
		case ActionCreateNewAndPromote:
			newVer, err := h.performCreateNew(ctx, keyName)
			if err != nil {
				merr = errors.Join(merr, err)
				continue
			}
			logger.InfoContext(ctx, "promoting immediately")
			if err := SetPrimary(ctx, h.kmsClient, keyName, newVer.GetName()); err != nil {
				merr = errors.Join(merr, err)
			}
		case ActionDisable:
			if err := h.performDisable(ctx, action.Version); err != nil {
				merr = errors.Join(merr, err)
				continue
			}
		case ActionDestroy:
			if err := h.performDestroy(ctx, action.Version); err != nil {
				merr = errors.Join(merr, err)
			}
		}
	}

	return
}

func (h *RotationHandler) performDisable(ctx context.Context, ver *kmspb.CryptoKeyVersion) error {
	logger := logging.FromContext(ctx)

	// Make a copy to modify
	newVerState := ver

	logger.InfoContext(ctx, "disabling key version", "version_name", ver.GetName())
	newVerState.State = kmspb.CryptoKeyVersion_DISABLED
	var messageType *kmspb.CryptoKeyVersion
	mask, err := fieldmaskpb.New(messageType, "state")
	if err != nil {
		return fmt.Errorf("failed to create fieldmask: %w", err)
	}
	updateReq := &kmspb.UpdateCryptoKeyVersionRequest{
		CryptoKeyVersion: newVerState,
		UpdateMask:       mask,
	}
	if _, err := h.kmsClient.UpdateCryptoKeyVersion(ctx, updateReq); err != nil {
		return fmt.Errorf("key disable failed: %w", err)
	}
	return nil
}

func (h *RotationHandler) performDestroy(ctx context.Context, ver *kmspb.CryptoKeyVersion) error {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "destroying key version", "version_name", ver.GetName())
	destroyReq := &kmspb.DestroyCryptoKeyVersionRequest{
		Name: ver.GetName(),
	}
	if _, err := h.kmsClient.DestroyCryptoKeyVersion(ctx, destroyReq); err != nil {
		return fmt.Errorf("key destroy failed: %w", err)
	}
	return nil
}

func (h *RotationHandler) performCreateNew(ctx context.Context, keyName string) (*kmspb.CryptoKeyVersion, error) {
	logger := logging.FromContext(ctx)
	logger.InfoContext(ctx, "creating new key version")

	createReq := &kmspb.CreateCryptoKeyVersionRequest{
		Parent:           keyName,
		CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
	}
	resp, err := h.kmsClient.CreateCryptoKeyVersion(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("key creation failed: %w", err)
	}

	// Wait for a key version to be created and enabled.
	var ckv *kmspb.CryptoKeyVersion
	b := retry.WithMaxRetries(5, retry.NewFibonacci(500*time.Millisecond))
	if err := retry.Do(ctx, b, func(ctx context.Context) error {
		var err error
		ckv, err = h.kmsClient.GetCryptoKeyVersion(ctx, &kmspb.GetCryptoKeyVersionRequest{
			Name: resp.GetName(),
		})
		if err != nil {
			return fmt.Errorf("failed to get crypto key version: %w", err)
		}
		if got, want := ckv.GetState(), kmspb.CryptoKeyVersion_ENABLED; got != want {
			return retry.RetryableError(fmt.Errorf("expected %s to be %s", got.String(), want.String()))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("key did not enter ready state: %w", err)
	}

	return ckv, nil
}

// GetKeyNameFromVersion converts a key version name to a key name.
//
// Example:
//
//	projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/* -> projects/*/locations/*/keyRings/*/cryptoKeys/*
func getKeyNameFromVersion(keyVersionName string) (string, error) {
	split := strings.Split(keyVersionName, "/")
	if len(split) != 10 {
		return "", fmt.Errorf("input had unexpected format: \"%s\"", keyVersionName)
	}
	// cut off last 2 values, re-combine
	return strings.Join(split[:len(split)-2], "/"), nil
}
