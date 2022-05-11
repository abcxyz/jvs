package jvscrypto

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"sort"

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

// JWKList creates a list of public keys in JWK format.
// https://datatracker.ietf.org/doc/html/rfc7517#section-4
func (k *KeyServer) JWKList(ctx context.Context, keyName string) ([]*jwk.Key, error) {
	states, err := k.StateStore.GetActiveVersionStates(ctx, keyName)
	if err != nil {
		return nil, fmt.Errorf("err while reading states: %w", err)
	}

	jwkList := make([]*jwk.Key, 0)

	for ver, _ := range states {
		key, err := k.KmsClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: ver})
		if err != nil {
			return nil, fmt.Errorf("err while getting public key from kms: %w", err)
		}

		block, _ := pem.Decode([]byte(key.Pem))
		if block == nil || block.Type != "PUBLIC KEY" {
			log.Fatal("failed to decode PEM block containing public key")
		}

		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			log.Fatal(err)
		}

		wk, err := jwk.New(pub)
		if err != nil {
			return nil, fmt.Errorf("err while converting public key to jwk: %w", err)
		}
		// TODO: We should set something else for Key ID. #27
		id, err := getLabelKey(ver)
		if err != nil {
			return nil, fmt.Errorf("err while determining key id: %w", err)
		}
		wk.Set(jwk.KeyIDKey, id)
		jwkList = append(jwkList, &wk)
	}
	sort.Slice(jwkList, func(i, j int) bool {
		return (*jwkList[i]).KeyID() < (*jwkList[j]).KeyID()
	})
	return jwkList, nil
}

// FormatJWKString creates a JWK Set converted to string.
// https://datatracker.ietf.org/doc/html/rfc7517#section-5
func FormatJWKString(wks []*jwk.Key) (string, error) {
	jwkMap := make(map[string][]*jwk.Key)
	jwkMap["keys"] = wks

	json, err := json.Marshal(jwkMap)
	if err != nil {
		return "", fmt.Errorf("err while converting jwk to json: %w", err)
	}
	return string(json), nil
}
