package jvscrypto

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"sort"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/pkg/cache"
	"github.com/abcxyz/jvs/pkg/config"
	"google.golang.org/api/iterator"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

// KeyServer provides all valid and active public keys in a JWKS format.
type KeyServer struct {
	KmsClient    *kms.KeyManagementClient
	CryptoConfig *config.CryptoConfig
	Cache        *cache.Cache[string]
}

type ECDSAKey struct {
	Curve string `json:"crv"`
	Id    string `json:"kid"`
	Type  string `json:"kty"`
	X     string `json:"x"`
	Y     string `json:"y"`
}

const cacheKey = "jwks"

// ServeHTTP returns the public keys in JWK format
func (k *KeyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	val, found := k.Cache.Lookup(cacheKey)
	if found {
		fmt.Fprintf(w, val)
		return
	}

	jwks := make([]*ECDSAKey, 0)
	for _, key := range k.CryptoConfig.KeyNames {
		list, err := k.JWKList(r.Context(), key)
		if err != nil {
			log.Printf("ran into error while determining public keys. %v\n", err)
			http.Error(w, "error determining public keys", http.StatusInternalServerError)
			return
		}
		jwks = append(jwks, list...)
	}
	json, err := FormatJWKString(jwks)
	if err != nil {
		log.Printf("ran into error while formatting public keys. %v\n", err)
		http.Error(w, "error formatting public keys", http.StatusInternalServerError)
		return
	}
	k.Cache.Set(cacheKey, json)
	fmt.Fprintf(w, json)
}

// JWKList creates a list of public keys in JWK format.
// https://datatracker.ietf.org/doc/html/rfc7517#section-4
func (k *KeyServer) JWKList(ctx context.Context, keyName string) ([]*ECDSAKey, error) {
	it := k.KmsClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: keyName,
		Filter: "state=ENABLED",
	})

	jwkList := make([]*ECDSAKey, 0)
	for {
		ver, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("err while reading crypto key version list: %w", err)
		}
		key, err := k.KmsClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: ver.Name})
		if err != nil {
			return nil, fmt.Errorf("err while getting public key from kms: %w", err)
		}

		block, _ := pem.Decode([]byte(key.Pem))
		if block == nil || block.Type != "PUBLIC KEY" {
			return nil, fmt.Errorf("failed to decode PEM block containing public key")
		}

		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key")
		}

		// TODO: We should set something else for Key ID. #27
		id, err := getLabelValue(ver.Name)
		if err != nil {
			return nil, fmt.Errorf("err while determining key id: %w", err)
		}

		ecdsaKey, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("Unknown key format")

		}
		ek := &ECDSAKey{
			Curve: "P-256",
			Id:    id,
			Type:  "EC",
			X:     base64.RawURLEncoding.EncodeToString(ecdsaKey.X.Bytes()),
			Y:     base64.RawURLEncoding.EncodeToString(ecdsaKey.Y.Bytes()),
		}
		jwkList = append(jwkList, ek)
	}
	sort.Slice(jwkList, func(i, j int) bool {
		return (*jwkList[i]).Id < (*jwkList[j]).Id
	})
	return jwkList, nil
}

// FormatJWKString creates a JWK Set converted to string.
// https://datatracker.ietf.org/doc/html/rfc7517#section-5
func FormatJWKString(wks []*ECDSAKey) (string, error) {
	jwkMap := make(map[string][]*ECDSAKey)
	jwkMap["keys"] = wks

	json, err := json.Marshal(jwkMap)
	if err != nil {
		return "", fmt.Errorf("err while converting jwk to json: %w", err)
	}
	return string(json), nil
}
