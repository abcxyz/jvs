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

	"google-on-gcp/jvs/services/go/apis/v1alpha1"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/api/iterator"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type RotationHandler struct {
	KmsClient    *kms.KeyManagementClient
	CryptoConfig v1alpha1.CryptoConfig
	// format: `projects/*/locations/*/keyRings/*/cryptoKeys/*`
	KeyName string
}

type PubSubMessage struct {
	Message struct {
		Action string `json:"action"`
	} `json:"message"`
}

// ConsumeMessage is called when a message is pushed to this server.
func (h *RotationHandler) ConsumeMessage(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received Message.")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var msg PubSubMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("json.Unmarshal: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Create the request to list Keys.
	listKeysReq := &kmspb.ListCryptoKeyVersionsRequest{
		Parent: h.KeyName,
	}

	// List the Keys in the KeyRing.
	it := h.KmsClient.ListCryptoKeyVersions(r.Context(), listKeysReq)
	actions, err := h.determineActions(it)
	if err != nil {
		log.Printf("Unable to determine cert actions: %v", err)
		http.Error(w, "Couldn't determine cert actions.", http.StatusInternalServerError)
		return
	}

	if err = h.performActions(r.Context(), actions); err != nil {
		log.Printf("Unable to perform cert actions: %v", err)
		http.Error(w, "Couldn't perform cert actions.", http.StatusInternalServerError)
		return
	}
}

type Action int64

const (
	NoAction Action = iota
	CreateNew
	Disable
	Destroy
)

// TODO: unclear if we need to handle import key use cases.
func (h *RotationHandler) determineActions(it *kms.CryptoKeyVersionIterator) (map[*kmspb.CryptoKeyVersion]Action, error) {
	// Older Key Version
	oldVers := map[*kmspb.CryptoKeyVersion]bool{}

	// Keep track of newest key version
	var newestEnabledVersion *kmspb.CryptoKeyVersion
	var newestTime int64

	for {
		ver, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		createTime := ver.CreateTime
		secondsOld := createTime.Seconds - time.Now().Unix()
		if secondsOld < newestTime && ver.State == kmspb.CryptoKeyVersion_ENABLED {
			oldVers[newestEnabledVersion] = true
			newestEnabledVersion = ver
			newestTime = secondsOld
		} else {
			oldVers[ver] = true
		}
	}

	actions := h.determineActionsForOlderVersions(oldVers)
	actions[newestEnabledVersion] = h.determineActionForNewestVersion(newestEnabledVersion)

	return actions, nil
}

// Determine whether the newest key needs to be rotated.
// The only actions available are NoAction and CreateNew. This is because we never
// want to disable/delete our newest key if we don't have a newer second one created.
func (h *RotationHandler) determineActionForNewestVersion(ver *kmspb.CryptoKeyVersion) Action {
	if ver == nil {
		log.Printf("!! Unable to find any enabled key version !!")
		// TODO: Do we want to fire a metric/other way to make this more visible?
		return CreateNew
	}

	secondsOld := ver.CreateTime.Seconds - time.Now().Unix()
	if int(secondsOld) > int(h.CryptoConfig.GetRotationAgeSeconds()) {
		log.Printf("Time to rotate newest key version.")
		return CreateNew
	}
	return NoAction
}

// This determines which action to take on key versions that are not the primary one (newest active).
// Since these aren't the primary key version, they can be disabled, or destroyed as long as sufficient time has passed.
func (h *RotationHandler) determineActionsForOlderVersions(vers map[*kmspb.CryptoKeyVersion]bool) map[*kmspb.CryptoKeyVersion]Action {
	actions := make(map[*kmspb.CryptoKeyVersion]Action)

	for ver := range vers {
		secondsOld := ver.CreateTime.Seconds - time.Now().Unix()
		switch ver.State {
		case kmspb.CryptoKeyVersion_ENABLED:
			if int(secondsOld) > int(h.CryptoConfig.GetDisableAgeSeconds()) {
				log.Printf("Key version %s will be disabled.", ver.Name)
				actions[ver] = Disable
			} else {
				log.Printf("Key version %s still within grace period, no action required.", ver.Name)
			}
		case kmspb.CryptoKeyVersion_DISABLED:
			if int(secondsOld) > int(h.CryptoConfig.GetDestroyAgeSeconds()) {
				log.Printf("Key version %s will be destroyed.", ver.Name)
				actions[ver] = Destroy
			} else {
				log.Printf("Key version %s still within disable period, no action required.", ver.Name)
			}
		case kmspb.CryptoKeyVersion_DESTROYED:
			log.Printf("Key version %s is already destroyed, no action required.", ver.Name)
		case kmspb.CryptoKeyVersion_DESTROY_SCHEDULED:
			log.Printf("Key version %s is scheduled to be destroyed, no action required.", ver.Name)
		case kmspb.CryptoKeyVersion_PENDING_IMPORT:
			// TODO: Would we ever import keys?
			log.Printf("Key version %s is being imported, no action required.", ver.Name)
		case kmspb.CryptoKeyVersion_IMPORT_FAILED:
			// TODO: do we want to error/metric on this? Would we ever import keys?
			log.Printf("Key version %s import failed! No automatic action can be performed.", ver.Name)
		}
	}
	return actions
}

func (h *RotationHandler) performActions(ctx context.Context, actions map[*kmspb.CryptoKeyVersion]Action) error {
	var result error
	for ver, action := range actions {
		switch action {
		case CreateNew:
			if err := h.performCreate(ctx, ver); err != nil {
				result = multierror.Append(result, err)
			}
		case Disable:
			if err := h.performDisable(ctx, ver); err != nil {
				result = multierror.Append(result, err)
			}
		case Destroy:
			if err := h.performDestroy(ctx, ver); err != nil {
				result = multierror.Append(result, err)
			}
		case NoAction:
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
	_, err = h.KmsClient.UpdateCryptoKeyVersion(ctx, updateReq)
	return err
}

func (h *RotationHandler) performDestroy(ctx context.Context, ver *kmspb.CryptoKeyVersion) error {
	log.Printf("Destroying Key Version %s", ver.Name)
	destroyReq := &kmspb.DestroyCryptoKeyVersionRequest{
		Name: ver.Name,
	}
	_, err := h.KmsClient.DestroyCryptoKeyVersion(ctx, destroyReq)
	return err
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
	_, err = h.KmsClient.CreateCryptoKeyVersion(ctx, createReq)
	return err
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
