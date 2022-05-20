package jvscrypto

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
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
	KMSClient       *kms.KeyManagementClient
	PublicKeyConfig *config.PublicKeyConfig
	Cache           *cache.Cache[string]
}

// ECDSAKey is the public key information for a Elliptic Curve Digital Signature Algorithm Key. used to serialize the public key
// into JWK format. https://datatracker.ietf.org/doc/html/rfc7517#section-4 .
type ECDSAKey struct {
	Curve string `json:"crv"`
	ID    string `json:"kid"`
	Type  string `json:"kty"`
	X     string `json:"x"`
	Y     string `json:"y"`
}

const cacheKey = "jwks"

// ServeHTTP returns the public keys in JWK format.
func (k *KeyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	val, err := k.Cache.WriteThruLookup(cacheKey, func() (string, error) {
		return k.generateJWKString(r.Context())
	})
	if err != nil {
		http.Error(w, "error generating jwk string", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, val)
}

func (k *KeyServer) generateJWKString(ctx context.Context) (string, error) {
	jwks := make([]*ECDSAKey, 0)
	for _, key := range k.PublicKeyConfig.KeyNames {
		list, err := k.jwkList(ctx, key)
		if err != nil {
			return "", fmt.Errorf("err while determining public keys %w", err)
		}
		jwks = append(jwks, list...)
	}
	json, err := formatJWKString(jwks)
	if err != nil {
		return "", fmt.Errorf("err while formatting public keys, %w", err)
	}
	return json, nil
}

// jwkList creates a list of public keys in JWK format.
// https://datatracker.ietf.org/doc/html/rfc7517#section-4 .
func (k *KeyServer) jwkList(ctx context.Context, keyName string) ([]*ECDSAKey, error) {
	it := k.KMSClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: keyName,
		Filter: "state=ENABLED",
	})

	jwkList := make([]*ECDSAKey, 0)
	for {
		// Could parallelize this. #34
		ver, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("err while reading crypto key version list: %w", err)
		}
		key, err := k.KMSClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: ver.Name})
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
			return nil, fmt.Errorf("unknown key format, expected ecdsa, got %T", pub)
		}
		if len(ecdsaKey.X.Bits()) == 0 || len(ecdsaKey.Y.Bits()) == 0 {
			return nil, fmt.Errorf("unable to determine X and/or Y for ECDSA key")
		}
		ek := &ECDSAKey{
			Curve: "P-256",
			ID:    id,
			Type:  "EC",
			X:     base64.RawURLEncoding.EncodeToString(ecdsaKey.X.Bytes()),
			Y:     base64.RawURLEncoding.EncodeToString(ecdsaKey.Y.Bytes()),
		}
		jwkList = append(jwkList, ek)
	}
	sort.Slice(jwkList, func(i, j int) bool {
		return (*jwkList[i]).ID < (*jwkList[j]).ID
	})
	return jwkList, nil
}

// formatJWKString creates a JWK Set converted to string.
// https://datatracker.ietf.org/doc/html/rfc7517#section-5 .
func formatJWKString(wks []*ECDSAKey) (string, error) {
	jwkMap := make(map[string][]*ECDSAKey)
	jwkMap["keys"] = wks

	json, err := json.Marshal(jwkMap)
	if err != nil {
		return "", fmt.Errorf("err while converting jwk to json: %w", err)
	}
	return string(json), nil
}
