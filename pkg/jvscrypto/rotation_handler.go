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
	CryptoConfig *config.CryptoConfig
	CurrentTime  time.Time
}

// RotateKey is called to determine and perform rotation actions on versions for a key.
// key is the full resource name: `projects/*/locations/*/keyRings/*/cryptoKeys/*`
// https://pkg.go.dev/google.golang.org/genproto/googleapis/cloud/kms/v1#CryptoKey
func (h *RotationHandler) RotateKey(ctx context.Context, key string) error {
	h.CurrentTime = time.Now()
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

	// Get any relevant Key Version information from the StateStore
	primaryName, err := GetPrimary(ctx, h.KmsClient, key)
	if err != nil {
		return fmt.Errorf("unable to determine primary: %w", err)
	}
	actions, err := h.determineActions(vers, primaryName)
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
	ActionCreateNew                  // New version should be created. Will be marked as new in StateStore (SS).
	ActionCreateNewAndPromote        // New version should be created. Will be marked as primary in SS.
	ActionPromote                    // Mark version as primary in SS.
	ActionDisable                    // Disable version. Will be removed from SS.
	ActionDestroy                    // Destroy version.
)

func (h *RotationHandler) determineActions(vers []*kmspb.CryptoKeyVersion, primaryName string) (map[*kmspb.CryptoKeyVersion]Action, error) {
	var primary *kmspb.CryptoKeyVersion
	var enabledVers []*kmspb.CryptoKeyVersion
	var disabledVers []*kmspb.CryptoKeyVersion
	var otherVers []*kmspb.CryptoKeyVersion

	for _, ver := range vers {
		log.Printf("checking version %v", ver)
		if primaryName != "" && ver.Name == primaryName {
			primary = ver
			continue
		}

		if ver.State == kmspb.CryptoKeyVersion_ENABLED {
			enabledVers = append(enabledVers, ver)
		} else if ver.State == kmspb.CryptoKeyVersion_DISABLED {
			disabledVers = append(disabledVers, ver)
		} else {
			otherVers = append(otherVers, ver)
		}
	}

	actions := h.actionsForDisabledVersions(disabledVers)

	enabledActions, promoting, err := h.actionsForEnabledVersions(enabledVers, primary)
	if err != nil {
		return nil, err
	}
	for k, v := range enabledActions {
		actions[k] = v
	}

	otherActions := h.actionsForOtherVersions(otherVers)
	if err != nil {
		return nil, err
	}
	for k, v := range otherActions {
		actions[k] = v
	}

	actions[primary] = h.actionsForPrimaryVersion(primary, promoting)

	return actions, nil
}

// Determine actions for non-primary enabled versions.
func (h *RotationHandler) actionsForEnabledVersions(enabledVers []*kmspb.CryptoKeyVersion, primary *kmspb.CryptoKeyVersion) (map[*kmspb.CryptoKeyVersion]Action, bool, error) {
	actions := make(map[*kmspb.CryptoKeyVersion]Action)
	promotingNewKey := false

	// if primary is nil, promote newest
	if primary == nil {
		var newest *kmspb.CryptoKeyVersion
		var newestTime time.Time

		for _, ver := range enabledVers {
			if newest == nil || ver.CreateTime.AsTime().After(newestTime) {
				newest = ver
				newestTime = ver.CreateTime.AsTime()
			}
		}

		if newest != nil {
			actions[newest] = ActionPromote
			return actions, true, nil
		}
		return actions, false, nil
	}

	for _, ver := range enabledVers {
		if ver.CreateTime.AsTime().After(primary.CreateTime.AsTime()) {
			// version is newer than primary, is a candidate for promotion.
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
			// version is older than primary, is a candidate for disabling.
			disableBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.KeyTTL)
			if ver.CreateTime.AsTime().Before(disableBeforeDate) {
				log.Printf("version %q created %q before cutoff date %q, will disable.\n", ver.Name, ver.CreateTime.AsTime(), disableBeforeDate)
				actions[ver] = ActionDisable
			} else {
				log.Printf("version %q created %q after disabled cutoff date %q, no action necessary.\n", ver.Name, ver.CreateTime.AsTime(), disableBeforeDate)
				actions[ver] = ActionNone
			}
		}
	}
	return actions, promotingNewKey, nil
}

// Determine actions for primary version.
func (h *RotationHandler) actionsForPrimaryVersion(primary *kmspb.CryptoKeyVersion, promotingNewKey bool) Action {
	if primary == nil && !promotingNewKey {
		log.Printf("no primary or new keys found, creating a new key and immediately promoting to primary.")
		return ActionCreateNewAndPromote
	}

	if promotingNewKey {
		log.Printf("promoting new key, no action on current primary.\n")
		return ActionNone
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

// Determine actions for disabled versions.
func (h *RotationHandler) actionsForDisabledVersions(vers []*kmspb.CryptoKeyVersion) map[*kmspb.CryptoKeyVersion]Action {
	actions := make(map[*kmspb.CryptoKeyVersion]Action)

	for _, ver := range vers {
		destroyBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.DestroyAge())
		if ver.CreateTime.AsTime().Before(destroyBeforeDate) {
			log.Printf("version %q created %q before cutoff date %q, will disable.\n", ver.Name, ver.CreateTime.AsTime(), destroyBeforeDate)
			actions[ver] = ActionDestroy
		} else {
			log.Printf("version %q created %q after cutoff date %q, no action necessary.\n", ver.Name, ver.CreateTime.AsTime(), destroyBeforeDate)
			actions[ver] = ActionNone
		}
	}
	return actions
}

// If a version is not enabled or disabled, determine actions.
func (h *RotationHandler) actionsForOtherVersions(vers []*kmspb.CryptoKeyVersion) map[*kmspb.CryptoKeyVersion]Action {
	actions := make(map[*kmspb.CryptoKeyVersion]Action)

	for _, ver := range vers {
		log.Printf("version %q in state %s, no action necessary.\n", ver.Name, ver.State.String())
		actions[ver] = ActionNone
	}
	return actions
}

// TODO: it may be worth adding rollback functionality for cases where multiple actions are expected to occur.
// for example, if we are demoting a key, we also need to ensure we've marked another as primary, and may end up
// in an odd state if one action occurs and the other does not.
func (h *RotationHandler) performActions(ctx context.Context, keyName string, actions map[*kmspb.CryptoKeyVersion]Action) error {
	var result error
	for ver, action := range actions {
		switch action {
		case ActionCreateNew:
			_, err := h.performCreateNew(ctx, keyName)
			if err != nil {
				result = multierror.Append(result, err)
				continue
			}
		case ActionPromote:
			if err := SetPrimary(ctx, h.KmsClient, keyName, ver.Name); err != nil {
				result = multierror.Append(result, err)
			}
		case ActionCreateNewAndPromote:
			newVer, err := h.performCreateNew(ctx, keyName)
			if err != nil {
				result = multierror.Append(result, err)
				continue
			}
			log.Printf("Promoting immediately.")
			if err := SetPrimary(ctx, h.KmsClient, keyName, newVer.Name); err != nil {
				result = multierror.Append(result, err)
			}
		case ActionDisable:
			if err := h.performDisable(ctx, ver); err != nil {
				result = multierror.Append(result, err)
				continue
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

func (h *RotationHandler) performCreateNew(ctx context.Context, keyName string) (*kmspb.CryptoKeyVersion, error) {
	log.Printf("creating new key version.")
	createReq := &kmspb.CreateCryptoKeyVersionRequest{
		Parent:           keyName,
		CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
	}
	resp, err := h.KmsClient.CreateCryptoKeyVersion(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("key creation failed: %w", err)
	}
	return resp, nil
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
