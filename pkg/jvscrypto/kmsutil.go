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
	"crypto"
	"errors"
	"fmt"
	"sort"
	"strings"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/abcxyz/pkg/workerpool"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const (
	PrimaryKey         = "primary"
	PrimaryLabelPrefix = "ver_"
)

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
		return fmt.Errorf("failed to create fieldmask: %w", err)
	}
	if _, err := kms.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{
		CryptoKey:  response,
		UpdateMask: mask,
	}); err != nil {
		return fmt.Errorf("failed to set labels: %w", err)
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

// CryptoKeyVersionsFor returns the list of cryptoKeyVersions for all the given
// parent keys.
func CryptoKeyVersionsFor(ctx context.Context, client *kms.KeyManagementClient, parentKeys []string) ([]string, error) {
	// Accumulate all the key versions for all provided keys.
	versionsWorker := workerpool.New[[]string](0)
	for _, parentKey := range parentKeys {
		parentKey := parentKey

		if err := versionsWorker.Do(ctx, func() ([]string, error) {
			it := client.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
				Parent: parentKey,
				Filter: "state=ENABLED",
			})

			var keyVersions []string
			for {
				keyVersion, err := it.Next()
				if errors.Is(err, iterator.Done) {
					break
				}
				if err != nil {
					return nil, fmt.Errorf("failed to get key version for %s: %w", parentKey, err)
				}
				keyVersions = append(keyVersions, keyVersion.Name)
			}
			return keyVersions, nil
		}); err != nil {
			return nil, fmt.Errorf("failed to get versions for %s: %w", parentKey, err)
		}
	}

	versionsResults, err := versionsWorker.Done(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}

	keyVersions := make(map[string]struct{}, len(versionsResults))
	for _, result := range versionsResults {
		if result.Error != nil {
			return nil, result.Error
		}

		for _, v := range result.Value {
			keyVersions[v] = struct{}{}
		}
	}

	final := make([]string, 0, len(keyVersions))
	for v := range keyVersions {
		final = append(final, v)
	}
	sort.Strings(final)
	return final, nil
}

// PublicKeysFor returns a map of a Cloud KMS key version name to the public key
// PEM for that key version for all the parent keys. It only returns keys that
// are enabled.
func PublicKeysFor(ctx context.Context, client *kms.KeyManagementClient, keyVersions []string) (map[string]crypto.PublicKey, error) {
	type keyPair struct {
		name      string
		publicKey crypto.PublicKey
	}

	publicKeysWorker := workerpool.New[*keyPair](0)
	for _, keyVersion := range keyVersions {
		keyVersion := keyVersion

		if err := publicKeysWorker.Do(ctx, func() (*keyPair, error) {
			publicKeyResp, err := client.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{
				Name: keyVersion,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to get public key for key version %s: %w", keyVersion, err)
			}

			publicKey, _, err := jwk.DecodePEM([]byte(publicKeyResp.Pem))
			if err != nil {
				return nil, fmt.Errorf("failed to decode pem for key version %s: %w", keyVersion, err)
			}
			return &keyPair{
				name:      keyVersion,
				publicKey: publicKey,
			}, nil
		}); err != nil {
			return nil, fmt.Errorf("failed to get public key for %s: %w", keyVersion, err)
		}
	}

	publicKeysResults, err := publicKeysWorker.Done(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}

	final := make(map[string]crypto.PublicKey, len(publicKeysResults))
	for _, result := range publicKeysResults {
		if result.Error != nil {
			return nil, fmt.Errorf("failed to fetch public key: %w", result.Error)
		}
		final[result.Value.name] = result.Value.publicKey
	}
	return final, nil
}

// JWKSet represents a set of JWK keys. The lestrrat-go/jwx/v2/jwk library has a
// jwk.Set, but it sorts keys by the key material, but we want to maintain our
// own, deterministic sort order. The jwk.Set is also an interface that is
// somewhat difficult to work with.
type JWKSet struct {
	Keys []jwk.Key `json:"keys"`
}

// JWKSFromPublicKeys converts the public keys to a JWK set. The keys are
// inserted in lexographical order by the key version name and returned in JSON
// format.
func JWKSFromPublicKeys(publicKeys map[string]crypto.PublicKey) (*JWKSet, error) {
	// Sort the list of key version names. This is largely for testing purposes,
	// since it creates a deterministic list of jwks.
	keyVersionNames := make([]string, 0, len(publicKeys))
	for k := range publicKeys {
		keyVersionNames = append(keyVersionNames, k)
	}
	sort.Strings(keyVersionNames)

	// Build the jwks
	jwkSet := &JWKSet{
		Keys: make([]jwk.Key, 0, len(keyVersionNames)),
	}
	for _, keyVersion := range keyVersionNames {
		publicKey := publicKeys[keyVersion]
		key, err := jwk.FromRaw(publicKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create jwk from public key for %s: %w", keyVersion, err)
		}

		if err := key.Set(jwk.KeyIDKey, keyVersion); err != nil {
			return nil, fmt.Errorf("failed to set kid %s on jwk: %w", keyVersion, err)
		}

		jwkSet.Keys = append(jwkSet.Keys, key)
	}

	return jwkSet, nil
}
