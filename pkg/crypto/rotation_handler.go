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

package crypto

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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
	// KeyName format: `projects/*/locations/*/keyRings/*/cryptoKeys/*`
	// https://pkg.go.dev/google.golang.org/genproto/googleapis/cloud/kms/v1#CryptoKey
	CurrentTime time.Time
}

// HTTPMessage is the request format we will send from the cloud scheduler
type HTTPMessage struct {
	Message struct {
		// TODO: We should support manual actions through call arguments, such as a rotation before the TTL. https://github.com/abcxyz/jvs/issues/9
		KeyName string
	} `json:"message"`
}

// ConsumeMessage is called when a message is pushed to this server.
func (h *RotationHandler) ConsumeMessage(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received Message.")
	// TODO: Use LimitReader. https://github.com/abcxyz/jvs/issues/7
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var msg HTTPMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("json.Unmarshal: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Create the request to list Keys.
	listKeysReq := &kmspb.ListCryptoKeyVersionsRequest{
		Parent: msg.Message.KeyName,
	}

	// List the Keys in the KeyRing.
	it := h.KmsClient.ListCryptoKeyVersions(r.Context(), listKeysReq)
	vers := make([]*kmspb.CryptoKeyVersion, 1)
	for {
		ver, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("Err while reading crypto key version list: %v", err)
			http.Error(w, "Err while reading crypto key version list.", http.StatusInternalServerError)
			return
		}
		vers = append(vers, ver)
	}

	actions, err := h.determineActions(vers)
	if err != nil {
		log.Printf("Unable to determine cert actions: %v", err)
		http.Error(w, "Couldn't determine cert actions.", http.StatusInternalServerError)
		return
	}

	if err = h.performActions(r.Context(), actions); err != nil {
		log.Printf("Unable to perform some cert actions: %v", err)
		http.Error(w, "Couldn't perform some cert actions.", http.StatusInternalServerError)
		return
	}
}

type Action int8

const (
	ActionNone Action = iota
	ActionCreate
	ActionDisable
	ActionDestroy
)

func (h *RotationHandler) determineActions(vers []*kmspb.CryptoKeyVersion) (map[*kmspb.CryptoKeyVersion]Action, error) {
	// Older Key Version
	otherVers := map[*kmspb.CryptoKeyVersion]struct{}{}

	// Keep track of newest key version
	var newestEnabledVersion *kmspb.CryptoKeyVersion
	var newestTime time.Time

	// Is there a key version currently in the process of being created.
	var newBeingGenerated = false

	for _, ver := range vers {
		if ver.State == kmspb.CryptoKeyVersion_ENABLED && (newestEnabledVersion == nil || ver.CreateTime.AsTime().After(newestTime)) {
			if newestEnabledVersion != nil {
				otherVers[newestEnabledVersion] = struct{}{}
			}
			newestEnabledVersion = ver
			newestTime = ver.CreateTime.AsTime()
		} else {
			if ver.State == kmspb.CryptoKeyVersion_PENDING_GENERATION || ver.State == kmspb.CryptoKeyVersion_PENDING_IMPORT {
				newBeingGenerated = true
			}
			otherVers[ver] = struct{}{}
		}
	}

	actions := h.actionsForOtherVersions(otherVers)
	actions[newestEnabledVersion] = h.actionForNewestVersion(newestEnabledVersion, newBeingGenerated)

	return actions, nil
}

// Determine whether the newest key needs to be rotated.
// The only actions available are ActionNone and ActionCreate. This is because we never
// want to disable/delete our newest key if we don't have a newer second one created.
func (h *RotationHandler) actionForNewestVersion(ver *kmspb.CryptoKeyVersion, newBeingGenerated bool) Action {
	if newBeingGenerated {
		log.Printf("Already have a new key being generated, no actions necessary")
		return ActionNone
	}
	if ver == nil {
		log.Printf("!! Unable to find any enabled key version !!")
		// TODO: Do we want to fire a metric/other way to make this more visible? https://github.com/abcxyz/jvs/issues/10
		return ActionCreate
	}

	rotateBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.GetRotationAge())
	if ver.CreateTime.AsTime().Before(rotateBeforeDate) {
		log.Printf("Time to rotate newest key version.")
		return ActionCreate
	}
	return ActionNone
}

// This determines which action to take on key versions that are not the primary one (newest active).
// Since these aren't the primary key version, they can be disabled, or destroyed as long as sufficient time has passed.
func (h *RotationHandler) actionsForOtherVersions(vers map[*kmspb.CryptoKeyVersion]struct{}) map[*kmspb.CryptoKeyVersion]Action {
	actions := make(map[*kmspb.CryptoKeyVersion]Action)

	for ver := range vers {
		switch ver.State {
		case kmspb.CryptoKeyVersion_ENABLED:
			disableBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.KeyTTL)
			if ver.CreateTime.AsTime().Before(disableBeforeDate) {
				log.Printf("Key version %s will be disabled.", ver.Name)
				actions[ver] = ActionDisable
			} else {
				log.Printf("Key version %s still within grace period, no action required.", ver.Name)
				actions[ver] = ActionNone
			}
		case kmspb.CryptoKeyVersion_DISABLED:
			destroyBeforeDate := h.CurrentTime.Add(-h.CryptoConfig.GetDestroyAge())
			if ver.CreateTime.AsTime().Before(destroyBeforeDate) {
				log.Printf("Key version %s will be destroyed.", ver.Name)
				actions[ver] = ActionDestroy
			} else {
				log.Printf("Key version %s still within disable period, no action required.", ver.Name)
				actions[ver] = ActionNone
			}
		default:
			// TODO: handle import cases. https://github.com/abcxyz/jvs/issues/5
			log.Printf("Key version state: %v. No action", ver.State)
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
	// Make a copy to modify
	newVerState := ver

	log.Printf("Disabling Key Version %s", ver.Name)
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
	log.Printf("Destroying Key Version %s", ver.Name)
	destroyReq := &kmspb.DestroyCryptoKeyVersionRequest{
		Name: ver.Name,
	}
	if _, err := h.KmsClient.DestroyCryptoKeyVersion(ctx, destroyReq); err != nil {
		return fmt.Errorf("key destroy failed: %w", err)
	}
	return nil
}

func (h *RotationHandler) performCreate(ctx context.Context, ver *kmspb.CryptoKeyVersion) error {
	log.Printf("Creating new Key Version.")
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
// -> `projects/*/locations/*/keyRings/*/cryptoKeys/*`
func getKeyNameFromVersion(keyVersionName string) (string, error) {
	split := strings.Split(keyVersionName, "/")
	if len(split) != 10 {
		return "", fmt.Errorf("input had unexpected format: \"%s\"", keyVersionName)
	}
	// cut off last 2 values, re-combine
	return strings.Join(split[:len(split)-2], "/"), nil
}
