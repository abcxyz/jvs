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
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/bigtable"
	"github.com/abcxyz/jvs/pkg/config"

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
	BTClient     *bigtable.Client
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

	// List the Key Versions in the Key
	it := h.KmsClient.ListCryptoKeyVersions(ctx, listKeysReq)
	vers := make([]*kmspb.CryptoKeyVersion, 0)
	for {
		ver, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("err while reading crypto key version list: %w", err)

		}
		vers = append(vers, ver)
	}

	// Get any relevant Key Version information from the BigTable
	states, err := GetActiveVersionStates(ctx, h.BTClient)
	if err != nil {
		return fmt.Errorf("err while reading big table: %w", err)
	}

	actions, err := h.determineActions(vers, states)
	if err != nil {
		return fmt.Errorf("unable to determine cert actions: %w", err)
	}

	if err = h.performActions(ctx, key, actions); err != nil {
		return fmt.Errorf("unable to perform some cert actions: %w", err)
	}
	return nil
}

type Action int8

const (
	ActionNone                Action = iota
	ActionCreateNew                  // New version should be created. Will be marked as new in BigTable (BT).
	ActionCreateNewAndPromote        // New version should be created. Will be marked as primary in BT.
	ActionPromote                    // Mark version as primary in BT.
	ActionDemote                     // Mark version as old in BT.
	ActionDisable                    // Disable version. Will be removed from BT.
	ActionDestroy                    // Destroy version.
)

func (h *RotationHandler) determineActions(vers []*kmspb.CryptoKeyVersion, activeStates map[string]VersionState) (map[*kmspb.CryptoKeyVersion]Action, error) {
	var primary *kmspb.CryptoKeyVersion
	var newVers []*kmspb.CryptoKeyVersion
	var oldVers []*kmspb.CryptoKeyVersion
	var inactiveVers []*kmspb.CryptoKeyVersion

	for _, ver := range vers {
		log.Printf("checking version %v", ver)
		if state, ok := activeStates[ver.Name]; ok {
			// Key is in the database, we consider it active.
			switch state {
			case VER_STATE_PRIMARY:
				primary = ver
			case VER_STATE_NEW:
				newVers = append(newVers, ver)
			case VER_STATE_OLD:
				oldVers = append(oldVers, ver)
			}
		} else {
			// Key isn't in the db, we consider it inactive.
			inactiveVers = append(inactiveVers, ver)
		}
	}

	actions := h.actionsForInactiveVersions(inactiveVers)
	activeActions, err := h.actionsForActiveVersions(primary, otherActiveVers)
	if err != nil {
		return nil, err
	}
	for k, v := range activeActions {
		actions[k] = v
	}

	return actions, nil
}

// Determine whether the newest key needs to be rotated.
// The only actions available are ActionNone and ActionCreate. This is because we never
// want to disable/delete our newest key if we don't have a newer second one created.
func (h *RotationHandler) actionsForNewVersions(newVers []*kmspb.CryptoKeyVersion) (map[*kmspb.CryptoKeyVersion]Action, bool, error) {
	actions := make(map[*kmspb.CryptoKeyVersion]Action)

	promotingNewKey := false
	for _, ver := range newVers {
		if ver.State == kmspb.CryptoKeyVersion_ENABLED {
			promoteBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.PropagationDelay)
			if ver.CreateTime.AsTime().Before(promoteBeforeDate) {
				log.Printf("version created %q before cutoff date %q, will promote to primary.\n", ver.CreateTime.AsTime(), promoteBeforeDate)
				actions[ver] = ActionPromote
				promotingNewKey = true
			} else {
				log.Printf("version created %q after cutoff date %q, no action necessary.\n", ver.CreateTime.AsTime(), promoteBeforeDate)
				actions[ver] = ActionNone
			}
		} else {
			log.Printf("version is disabled, no action necessary.\n")
			actions[ver] = ActionNone
		}
	}
	return actions, promotingNewKey, nil
}

func (h *RotationHandler) actionsForOldVersions(oldVers []*kmspb.CryptoKeyVersion) (map[*kmspb.CryptoKeyVersion]Action, error) {
	actions := make(map[*kmspb.CryptoKeyVersion]Action)

	for _, ver := range oldVers {
		disableBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.KeyTTL)
		if ver.CreateTime.AsTime().Before(disableBeforeDate) {
			log.Printf("version %q created %q before cutoff date %q, will disable.\n", ver.Name, ver.CreateTime.AsTime(), disableBeforeDate)
			actions[ver] = ActionDisable
		} else {
			log.Printf("version %q created %q after disabled cutoff date %q, no action necessary.\n", ver.Name, ver.CreateTime.AsTime(), disableBeforeDate)
			actions[ver] = ActionNone
		}
	}
	return actions, nil
}

func (h *RotationHandler) actionsForPrimaryVersion(primary *kmspb.CryptoKeyVersion, promotingNewKey bool) Action {
	if primary == nil && !promotingNewKey {
		log.Printf("no primary or new keys found, creating a new key and immediately promoting to primary.")
		return ActionCreateNewAndPromote
	}

	// We're promoting another key to primary, demote the current primary.
	if promotingNewKey {
		return ActionDemote
	}

	// We don't have a new key we're promoting, see if we should create a new key.
	rotateBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.RotationAge())
	if primary.CreateTime.AsTime().Before(rotateBeforeDate) {
		log.Printf("version created %q before cutoff date %q, will rotate.\n", primary.CreateTime.AsTime(), rotateBeforeDate)
		return ActionCreateNew
	} else {
		log.Printf("version created %q after cutoff date %q, no action necessary.\n", primary.CreateTime.AsTime(), rotateBeforeDate)
		return ActionNone
	}
}

// This determines which action to take on key versions that are not the primary one (newest active).
// Since these aren't the primary key version, they can be disabled, or destroyed as long as sufficient time has passed.
func (h *RotationHandler) actionsForInactiveVersions(vers []*kmspb.CryptoKeyVersion) map[*kmspb.CryptoKeyVersion]Action {
	actions := make(map[*kmspb.CryptoKeyVersion]Action)

	for _, ver := range vers {
		if ver.State == kmspb.CryptoKeyVersion_DISABLED {
			destroyBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.DestroyAge())
			if ver.CreateTime.AsTime().Before(destroyBeforeDate) {
				log.Printf("version %q created %q before cutoff date %q, will disable.\n", ver.Name, ver.CreateTime.AsTime(), destroyBeforeDate)
				actions[ver] = ActionDestroy
			} else {
				log.Printf("version %q created %q after cutoff date %q, no action necessary.\n", ver.Name, ver.CreateTime.AsTime(), destroyBeforeDate)
				actions[ver] = ActionNone
			}
		} else {
			// TODO: handle import cases. https://github.com/abcxyz/jvs/issues/5
			log.Printf("key version in state: %v. No action necessary.", ver.State)
			actions[ver] = ActionNone
		}
	}
	return actions
}

func (h *RotationHandler) performActions(ctx context.Context, keyName string, actions map[*kmspb.CryptoKeyVersion]Action) error {
	var result error
	for ver, action := range actions {
		switch action {
		case ActionCreateNew:
			if err := h.performCreateNew(ctx, keyName); err != nil {
				result = multierror.Append(result, err)
			} else {
				if err := h.setState(ctx, ver, VER_STATE_NEW); err != nil {
					result = multierror.Append(result, err)
				}
			}
		case ActionPromote:
			if err := h.setState(ctx, ver, VER_STATE_PRIMARY); err != nil {
				result = multierror.Append(result, err)
			}
		case ActionCreateNewAndPromote:
			if err := h.performCreateNew(ctx, keyName); err != nil {
				result = multierror.Append(result, err)
			} else {
				if err := h.setState(ctx, ver, VER_STATE_PRIMARY); err != nil {
					result = multierror.Append(result, err)
				}
			}
		case ActionDisable:
			if err := h.performDisable(ctx, ver); err != nil {
				result = multierror.Append(result, err)
			} else {
				if err := RemoveVersionState(ctx, h.BTClient, ver.Name); err != nil {
					result = multierror.Append(result, err)
				}
			}
		case ActionDestroy:
			if err := h.performDestroy(ctx, ver); err != nil {
				result = multierror.Append(result, err)
			}
		case ActionDemote:
			if err := h.setState(ctx, ver, VER_STATE_OLD); err != nil {
				result = multierror.Append(result, err)
			}
		case ActionNone:
			continue
		}
	}
	return result
}

func (h *RotationHandler) performDisable(ctx context.Context, ver *kmspb.CryptoKeyVersion) error {
	// Make a copy to modify
	newVerState := ver

	log.Printf("disabling key version %s", ver.Name)
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
	log.Printf("destroying key version %s", ver.Name)
	destroyReq := &kmspb.DestroyCryptoKeyVersionRequest{
		Name: ver.Name,
	}
	if _, err := h.KmsClient.DestroyCryptoKeyVersion(ctx, destroyReq); err != nil {
		return fmt.Errorf("key destroy failed: %w", err)
	}
	return nil
}

func (h *RotationHandler) performCreateNew(ctx context.Context, keyName string) error {
	log.Printf("creating new key version.")
	createReq := &kmspb.CreateCryptoKeyVersionRequest{
		Parent:           keyName,
		CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
	}
	if _, err := h.KmsClient.CreateCryptoKeyVersion(ctx, createReq); err != nil {
		return fmt.Errorf("key creation failed: %w", err)
	}
	return nil
}

func (h *RotationHandler) setState(ctx context.Context, ver *kmspb.CryptoKeyVersion, state VersionState) error {
	return WriteVersionState(ctx, h.BTClient, ver.Name, state)
}

// GetKeyNameFromVersion converts a key version name to a key name.
// Example:
// `projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*`
// -> `projects/*/locations/*/keyRings/*/cryptoKeys/*`
func getKeyNameFromVersion(keyVersionName string) (string, error) {
	split := strings.Split(keyVersionName, "/")
	if len(split) != 10 {
		return "", fmt.Errorf("input had unexpected format: \"%s\"", keyVersionName)
	}
	// cut off last 2 values, re-combine
	return strings.Join(split[:len(split)-2], "/"), nil
}
