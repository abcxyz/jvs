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

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"

	jvspb "github.com/abcxyz/jvs/apis/v0"
)

// CertificateActionService allows for performing manual actions on certificate
// versions.
type CertificateActionService struct {
	jvspb.CertificateActionServiceServer
	Handler   *RotationHandler
	KMSClient *kms.KeyManagementClient
}

// CertificateAction implements the certificate action API which performs manual
// actions on cert versions. this wraps certificateAction and adds a blank
// response.
func (p *CertificateActionService) CertificateAction(ctx context.Context, request *jvspb.CertificateActionRequest) (*jvspb.CertificateActionResponse, error) {
	return &jvspb.CertificateActionResponse{}, p.certificateAction(ctx, request)
}

func (p *CertificateActionService) certificateAction(ctx context.Context, request *jvspb.CertificateActionRequest) error {
	// create map of key -> version actions list
	actions := make(map[string][]*actionTuple)
	for _, action := range request.GetActions() {
		key, err := getKeyNameFromVersion(action.GetVersion())
		if err != nil {
			return fmt.Errorf("couldn't determine key name from version %s: %w", action.GetVersion(), err)
		}
		keyActions, ok := actions[key]
		if !ok {
			keyActions = make([]*actionTuple, 0)
		}

		ver, err := p.KMSClient.GetCryptoKeyVersion(ctx, &kmspb.GetCryptoKeyVersionRequest{Name: action.GetVersion()})
		if err != nil {
			return fmt.Errorf("couldn't get key version %s: %w", action.GetVersion(), err)
		}

		primary, err := GetPrimary(ctx, p.KMSClient, key)
		if err != nil {
			return fmt.Errorf("couldn't determine current primary: %w", err)
		}

		keyActions = append(keyActions, determineActions(ver, action.GetAction(), primary)...)
		actions[key] = keyActions
	}

	for key, actionTuples := range actions {
		if err := p.Handler.performActions(ctx, key, actionTuples); err != nil {
			// If any actions fail, short circuit.
			return fmt.Errorf("couldn't perform actions %v on key %s: %w", actionTuples, key, err)
		}
	}
	return nil
}

// determineActions decides which changes we should make based on the asked for
// action, and current primary.
func determineActions(ver *kmspb.CryptoKeyVersion, action jvspb.Action_ACTION, primary string) []*actionTuple {
	actionsToPerform := make([]*actionTuple, 0)
	if primary == ver.GetName() {
		// We are modifying the current primary, we should create a new version and
		// immediately promote it.
		actionsToPerform = append(actionsToPerform, &actionTuple{
			Action: ActionCreateNewAndPromote,
		})
	}

	// See if any additional forced actions are necessary. If we specified rotate
	// as the action, then there is no additional actions to be taken. If that
	// version is primary, we have already created a new key and set the new key
	// as primary. If we have specified we also want to force disable or destroy
	// the key, there are additional actions that need to be taken.
	if action == jvspb.Action_FORCE_DISABLE {
		actionsToPerform = append(actionsToPerform, &actionTuple{
			Action:  ActionDisable,
			Version: ver,
		})
	} else if action == jvspb.Action_FORCE_DESTROY {
		actionsToPerform = append(actionsToPerform, &actionTuple{
			Action:  ActionDestroy,
			Version: ver,
		})
	}
	return actionsToPerform
}
