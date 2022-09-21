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
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"google.golang.org/api/iterator"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const (
	PrimaryKey         = "primary"
	PrimaryLabelPrefix = "ver_"
)

// JWKS represents a JWK Set, used to convert to json representation.
// https://datatracker.ietf.org/doc/html/rfc7517#section-5 .
type JWKS struct {
	Keys []*ECDSAKey `json:"keys"`
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

// GetLatestKeyVersion looks up the newest enabled key version. If there is no enabled version, this returns nil.
func GetLatestKeyVersion(ctx context.Context, kms *kms.KeyManagementClient, keyName string) (*kmspb.CryptoKeyVersion, error) {
	it := kms.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: keyName,
		Filter: "state=ENABLED",
	})

	var newestEnabledVersion *kmspb.CryptoKeyVersion
	var newestTime time.Time
	for {
		ver, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("err while reading crypto key version list: %w", err)
		}
		if newestEnabledVersion == nil || ver.CreateTime.AsTime().After(newestTime) {
			newestEnabledVersion = ver
			newestTime = ver.CreateTime.AsTime()
		}
	}
	return newestEnabledVersion, nil
}

// PublicKey returns the public key for the newest enabled key version.
func PublicKey(ctx context.Context, kms *kms.KeyManagementClient, keyName string) ([]byte, error) {
	ver, err := GetLatestKeyVersion(ctx, kms, keyName)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}
	// Retrieve the public key from KMS.
	response, err := kms.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: ver.Name})
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}
	return []byte(response.Pem), nil
}

// GetPrimary gets the key version name marked as primary in the key labels.
func GetPrimary(ctx context.Context, kms *kms.KeyManagementClient, key string) (string, error) {
	response, err := kms.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{Name: key})
	if err != nil {
		return "", fmt.Errorf("issue while getting key from KMS: %w", err)
	}
	if primary, ok := response.Labels[PrimaryKey]; ok {
		primary = strings.TrimPrefix(primary, PrimaryLabelPrefix)
		return fmt.Sprintf("%s/cryptoKeyVersions/%s", key, primary), nil
	}
	// no primary found
	return "", nil
}

// SetPrimary sets the key version name as primary in the key labels.
// 'Primary' field will be omitted for keys with purpose other than ENCRYPT_DECRYPT(https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys).
// Therefore, use `Labels` filed to set the primary key version name with format `ver_[CRYPTO_KEY_Version_ID]`.
// For example, "ver_1".
func SetPrimary(ctx context.Context, kms *kms.KeyManagementClient, key, versionName string) error {
	response, err := kms.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{Name: key})
	if err != nil {
		return fmt.Errorf("issue while getting key from KMS: %w", err)
	}

	value, err := getLabelValue(versionName)
	if err != nil {
		return err
	}
	// update label
	labels := response.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[PrimaryKey] = value
	response.Labels = labels

	var messageType *kmspb.CryptoKey
	mask, err := fieldmaskpb.New(messageType, "labels")
	if err != nil {
		return err
	}
	_, err = kms.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{CryptoKey: response, UpdateMask: mask})
	if err != nil {
		return fmt.Errorf("issue while setting labels in kms %w", err)
	}
	return nil
}

// This returns the key version name with "ver_" prefixed. This is because labels must start with a lowercase letter, and can't go over 64 chars.
// Example:  projects/*/locations/location1/keyRings/keyring1/cryptoKeys/key1/cryptoKeyVersions/1 -> ver_1 .
func getLabelValue(versionName string) (string, error) {
	split := strings.Split(versionName, "/")
	if len(split) != 10 {
		return "", fmt.Errorf("input had unexpected format: \"%s\"", versionName)
	}
	versionValue := PrimaryLabelPrefix + split[len(split)-1]
	return versionValue, nil
}

// JWKList creates a list of public keys in JWK format.
// https://datatracker.ietf.org/doc/html/rfc7517#section-4 .
func JWKList(ctx context.Context, kms *kms.KeyManagementClient, keyName string) ([]*ECDSAKey, error) {
	it := kms.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
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

		key, err := kms.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: ver.Name})
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

		ecdsaKey, ok := pub.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("unknown key format, expected ecdsa, got %T", pub)
		}
		if len(ecdsaKey.X.Bits()) == 0 || len(ecdsaKey.Y.Bits()) == 0 {
			return nil, fmt.Errorf("unable to determine X and/or Y for ECDSA key")
		}
		ek := &ECDSAKey{
			Curve: "P-256",
			ID:    ver.Name,
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

// FormatJWKString creates a JWK Set converted to string.
// https://datatracker.ietf.org/doc/html/rfc7517#section-5 .
func FormatJWKString(wks []*ECDSAKey) (string, error) {
	jwks := &JWKS{Keys: wks}
	json, err := json.Marshal(jwks)
	if err != nil {
		return "", fmt.Errorf("err while converting jwk to json: %w", err)
	}
	return string(json), nil
}
