package jvscrypto

import (
	"context"
	"encoding/json"
	"fmt"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/lestrrat-go/jwx/jwk"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

// KeyServer provides all valid and active public keys in a JWKS format.
type KeyServer struct {
	KmsClient    *kms.KeyManagementClient
	CryptoConfig *config.CryptoConfig
	StateStore   StateStore
}

// Creates a JWK Set converted to string.
// https://datatracker.ietf.org/doc/html/rfc7517#section-5
func (k *KeyServer) getJWKSetFormattedString(ctx context.Context, keyName string) (string, error) {
	wks, err := k.getJWKList(ctx, keyName)
	if err != nil {
		return "", err
	}
	jwkMap := make(map[string][]*jwk.Key)
	jwkMap["keys"] = wks

	json, err := json.MarshalIndent(jwkMap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("err while converting jwk to json: %w", err)
	}
	return string(json), nil
}

// Creates a list of public keys in JWK format.
// https://datatracker.ietf.org/doc/html/rfc7517#section-4
func (k *KeyServer) getJWKList(ctx context.Context, keyName string) ([]*jwk.Key, error) {
	states, err := k.StateStore.GetActiveVersionStates(ctx, keyName)
	if err != nil {
		return nil, fmt.Errorf("err while reading states: %w", err)
	}

	var jwkList []*jwk.Key

	for ver, _ := range states {
		key, err := k.KmsClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: ver})
		if err != nil {
			return nil, fmt.Errorf("err while getting public key from kms: %w", err)
		}
		wk, err := jwk.New(key.Pem)
		if err != nil {
			return nil, fmt.Errorf("err while converting public key to jwk: %w", err)
		}
		// TODO: Should we have something else for key id?
		id, err := getLabelKey(ver)
		if err != nil {
			return nil, fmt.Errorf("err while determining key id: %w", err)
		}
		wk.Set(jwk.KeyIDKey, id)
		jwkList = append(jwkList, &wk)
	}
	return jwkList, nil
}
