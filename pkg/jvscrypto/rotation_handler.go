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

	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/zlogger"
	"go.uber.org/zap"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/api/iterator"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// RotationHandler handles all necessary rotation actions for asymmetric keys based off a provided
// configuration.
type RotationHandler struct {
	KmsClient    *kms.KeyManagementClient
	CryptoConfig *config.CryptoConfig
	CurrentTime  time.Time
}

// RotateKey is called to determine and perform rotation actions on versions for a key.
// key is the full resource name: `projects/*/locations/*/keyRings/*/cryptoKeys/*`
// https://pkg.go.dev/google.golang.org/genproto/googleapis/cloud/kms/v1#CryptoKey
func (h *RotationHandler) RotateKey(ctx context.Context, key string) error {
	// Create the request to list Keys.
	listKeysReq := &kmspb.ListCryptoKeyVersionsRequest{
		Parent: key,
	}

	// List the Keys in the KeyRing.
	it := h.KmsClient.ListCryptoKeyVersions(ctx, listKeysReq)
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

	actions, err := h.determineActions(ctx, vers)
	if err != nil {
		return fmt.Errorf("unable to determine cert actions: %w", err)
	}

	if err = h.performActions(ctx, actions); err != nil {
		return fmt.Errorf("unable to perform some cert actions: %w", err)
	}
	return nil
}

type Action int8

const (
	ActionNone Action = iota
	ActionCreate
	ActionDisable
	ActionDestroy
)

func (h *RotationHandler) determineActions(ctx context.Context, vers []*kmspb.CryptoKeyVersion) (map[*kmspb.CryptoKeyVersion]Action, error) {
	logger := zlogger.FromContext(ctx)
	// Older Key Version
	var otherVers []*kmspb.CryptoKeyVersion

	// Keep track of newest key version
	var newestEnabledVersion *kmspb.CryptoKeyVersion
	var newestTime time.Time

	// Is there a key version currently in the process of being created.
	newBeingGenerated := false

	for _, ver := range vers {
		logger.Debugf("checking version", zap.Any("version", ver))
		if ver.State == kmspb.CryptoKeyVersion_ENABLED && (newestEnabledVersion == nil || ver.CreateTime.AsTime().After(newestTime)) {
			if newestEnabledVersion != nil {
				otherVers = append(otherVers, newestEnabledVersion)
			}
			newestEnabledVersion = ver
			newestTime = ver.CreateTime.AsTime()
		} else {
			if ver.State == kmspb.CryptoKeyVersion_PENDING_GENERATION || ver.State == kmspb.CryptoKeyVersion_PENDING_IMPORT {
				newBeingGenerated = true
			}
			otherVers = append(otherVers, ver)
		}
	}

	actions := h.actionsForOtherVersions(ctx, otherVers)
	actions[newestEnabledVersion] = h.actionForNewestVersion(ctx, newestEnabledVersion, newBeingGenerated)

	return actions, nil
}

// Determine whether the newest key needs to be rotated.
// The only actions available are ActionNone and ActionCreate. This is because we never
// want to disable/delete our newest key if we don't have a newer second one created.
func (h *RotationHandler) actionForNewestVersion(ctx context.Context, ver *kmspb.CryptoKeyVersion, newBeingGenerated bool) Action {
	logger := zlogger.FromContext(ctx)
	if newBeingGenerated {
		logger.Infof("already have a new key being generated, no actions necessary")
		return ActionNone
	}
	if ver == nil {
		logger.Errorf("!! unable to find any enabled key version !!")
		// TODO: Do we want to fire a metric/other way to make this more visible? https://github.com/abcxyz/jvs/issues/10
		return ActionCreate
	}

	rotateBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.RotationAge())
	if ver.CreateTime.AsTime().Before(rotateBeforeDate) {
		logger.Info("version created before cutoff date, will rotate.", zap.Any("version", ver), zap.Any("rotateBeforeDate", rotateBeforeDate))
		return ActionCreate
	}
	logger.Debug("version created before cutoff date, no action necessary.", zap.Any("version", ver), zap.Any("rotateBeforeDate", rotateBeforeDate))
	return ActionNone
}

// This determines which action to take on key versions that are not the primary one (newest active).
// Since these aren't the primary key version, they can be disabled, or destroyed as long as sufficient time has passed.
func (h *RotationHandler) actionsForOtherVersions(ctx context.Context, vers []*kmspb.CryptoKeyVersion) map[*kmspb.CryptoKeyVersion]Action {
	logger := zlogger.FromContext(ctx)

	actions := make(map[*kmspb.CryptoKeyVersion]Action)

	for _, ver := range vers {
		//nolint:exhaustive // TODO: handle import cases. https://github.com/abcxyz/jvs/issues/5
		switch ver.State {
		case kmspb.CryptoKeyVersion_ENABLED:
			disableBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.KeyTTL)
			if ver.CreateTime.AsTime().Before(disableBeforeDate) {
				logger.Info("version created before cutoff date, will disable.", zap.Any("version", ver), zap.Any("disableBeforeDate", disableBeforeDate))
				actions[ver] = ActionDisable
			} else {
				logger.Info("version created after cutoff date, no action necessary.", zap.Any("version", ver), zap.Any("disableBeforeDate", disableBeforeDate))
				actions[ver] = ActionNone
			}
		case kmspb.CryptoKeyVersion_DISABLED:
			destroyBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.DestroyAge())
			if ver.CreateTime.AsTime().Before(destroyBeforeDate) {
				logger.Info("version created before cutoff date, will destroy.", zap.Any("version", ver), zap.Any("destroyBeforeDate", destroyBeforeDate))
				actions[ver] = ActionDestroy
			} else {
				logger.Info("version created after cutoff date, no action necessary.", zap.Any("version", ver), zap.Any("destroyBeforeDate", destroyBeforeDate))
				actions[ver] = ActionNone
			}
		default:
			logger.Info("no action needed for key version in current state.", zap.Any("version", ver), zap.Any("state", ver.State))
			actions[ver] = ActionNone
		}
	}
	return actions
}

func (h *RotationHandler) performActions(ctx context.Context, actions map[*kmspb.CryptoKeyVersion]Action) error {
	var result error
	for ver, action := range actions {
		switch action {
		case ActionCreate:
			if err := h.performCreate(ctx, ver); err != nil {
				result = multierror.Append(result, err)
			}
		case ActionDisable:
			if err := h.performDisable(ctx, ver); err != nil {
				result = multierror.Append(result, err)
			}
		case ActionDestroy:
			if err := h.performDestroy(ctx, ver); err != nil {
				result = multierror.Append(result, err)
			}
		case ActionNone:
			continue
		}
	}
	return result
}

func (h *RotationHandler) performDisable(ctx context.Context, ver *kmspb.CryptoKeyVersion) error {
	logger := zlogger.FromContext(ctx)

	// Make a copy to modify
	newVerState := ver

	logger.Info("disabling key version", zap.String("versionName", ver.Name))
	newVerState.State = kmspb.CryptoKeyVersion_DISABLED
	var messageType *kmspb.CryptoKeyVersion
	mask, err := fieldmaskpb.New(messageType, "state")
	if err != nil {
		return err
	}
	updateReq := &kmspb.UpdateCryptoKeyVersionRequest{
		CryptoKeyVersion: newVerState,
		UpdateMask:       mask,
	}
	if _, err := h.KmsClient.UpdateCryptoKeyVersion(ctx, updateReq); err != nil {
		return fmt.Errorf("key disable failed: %w", err)
	}
	return nil
}

func (h *RotationHandler) performDestroy(ctx context.Context, ver *kmspb.CryptoKeyVersion) error {
	logger := zlogger.FromContext(ctx)
	logger.Info("destroying key version", zap.String("versionName", ver.Name))
	destroyReq := &kmspb.DestroyCryptoKeyVersionRequest{
		Name: ver.Name,
	}
	if _, err := h.KmsClient.DestroyCryptoKeyVersion(ctx, destroyReq); err != nil {
		return fmt.Errorf("key destroy failed: %w", err)
	}
	return nil
}

func (h *RotationHandler) performCreate(ctx context.Context, ver *kmspb.CryptoKeyVersion) error {
	logger := zlogger.FromContext(ctx)
	logger.Info("creating new key version.")
	key, err := getKeyNameFromVersion(ver.Name)
	if err != nil {
		return err
	}
	createReq := &kmspb.CreateCryptoKeyVersionRequest{
		Parent:           key,
		CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
	}
	if _, err = h.KmsClient.CreateCryptoKeyVersion(ctx, createReq); err != nil {
		return fmt.Errorf("key creation failed: %w", err)
	}
	return nil
}

// GetKeyNameFromVersion converts a key version name to a key name.
// Example:
// `projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*`
// -> `projects/*/locations/*/keyRings/*/cryptoKeys/*`.
func getKeyNameFromVersion(keyVersionName string) (string, error) {
	split := strings.Split(keyVersionName, "/")
	if len(split) != 10 {
		return "", fmt.Errorf("input had unexpected format: \"%s\"", keyVersionName)
	}
	// cut off last 2 values, re-combine
	return strings.Join(split[:len(split)-2], "/"), nil
}
