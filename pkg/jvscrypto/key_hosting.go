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
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/renderer"
)

// KeyServer provides all valid and active public keys in a JWKS format.
type KeyServer struct {
	kmsClient *kms.KeyManagementClient
	config    *config.PublicKeyConfig
	cache     *cache.Cache[string]
	h         *renderer.Renderer
}

// NewKeyServer creates a new server. See [KeyServer] for more information.
func NewKeyServer(ctx context.Context, kmsClient *kms.KeyManagementClient, cfg *config.PublicKeyConfig, h *renderer.Renderer) *KeyServer {
	cache := cache.New[string](cfg.CacheTimeout)

	return &KeyServer{
		kmsClient: kmsClient,
		config:    cfg,
		cache:     cache,
		h:         h,
	}
}

const cacheKey = "jwks"

// ServeHTTP returns the public keys in JWK format.
func (k *KeyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	val, err := k.cache.WriteThruLookup(cacheKey, func() (string, error) {
		return k.generateJWKString(r.Context())
	})
	if err != nil {
		logger.Errorw("error generating jwk string", "error", err)
		k.h.RenderJSON(w, http.StatusInternalServerError, fmt.Errorf("failed to generate jwks"))
		return
	}

	w.Header().Set("content-type", "application/json")
	fmt.Fprint(w, val)
}

func (k *KeyServer) generateJWKString(ctx context.Context) (string, error) {
	keyVersions, err := CryptoKeyVersionsFor(ctx, k.kmsClient, k.config.KeyNames)
	if err != nil {
		return "", fmt.Errorf("failed to list crypto keys: %w", err)
	}

	publicKeys, err := PublicKeysFor(ctx, k.kmsClient, keyVersions)
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
