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
	"encoding/json"
	"fmt"
	"net/http"

	kms "cloud.google.com/go/kms/apiv1"

	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/pkg/cache"
	logging "github.com/abcxyz/pkg/logging/exp"
)

// KeyServer provides all valid and active public keys in a JWKS format.
type KeyServer struct {
	KMSClient       *kms.KeyManagementClient
	PublicKeyConfig *config.PublicKeyConfig
	Cache           *cache.Cache[string]
}

const cacheKey = "jwks"

// ServeHTTP returns the public keys in JWK format.
func (k *KeyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	val, err := k.Cache.WriteThruLookup(cacheKey, func() (string, error) {
		return k.generateJWKString(r.Context())
	})
	if err != nil {
		logger.Error("error generating jwk string", "error", err)
		http.Error(w, "error generating jwk string", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, val)
}

func (k *KeyServer) generateJWKString(ctx context.Context) (string, error) {
	keyVersions, err := CryptoKeyVersionsFor(ctx, k.KMSClient, k.PublicKeyConfig.KeyNames)
	if err != nil {
		return "", fmt.Errorf("failed to list crypto keys: %w", err)
	}

	publicKeys, err := PublicKeysFor(ctx, k.KMSClient, keyVersions)
	if err != nil {
		return "", fmt.Errorf("failed to get public keys: %w", err)
	}

	jwks, err := JWKSFromPublicKeys(publicKeys)
	if err != nil {
		return "", fmt.Errorf("failed to create jwks: %w", err)
	}

	b, err := json.Marshal(jwks)
	if err != nil {
		return "", fmt.Errorf("failed to marshal jwks as json: %w", err)
	}
	return string(b), nil
}
