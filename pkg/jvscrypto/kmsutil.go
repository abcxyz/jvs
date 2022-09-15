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
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	jwt2 "github.com/lestrrat-go/jwx/v2/jwt"
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

// VerifyJWTString verifies that a JWT string is signed correctly and is valid.
func VerifyJWTString(ctx context.Context, kms *kms.KeyManagementClient, keyName string, jwtStr string) error {
	key, err := PublicKey(ctx, kms, keyName)
	if err != nil {
		return err
	}
	parsedKey, err := jwt.ParseECPublicKeyFromPEM(key)
	if err != nil {
		return fmt.Errorf("unable to parse key. %w", err)
	}

	parts := strings.Split(jwtStr, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid jwt string %s", jwtStr)
	}
	if err := jwt.SigningMethodES256.Verify(strings.Join(parts[0:2], "."), parts[2], parsedKey); err != nil {
		return fmt.Errorf("unable to verify signed jwt string. %w", err)
	}
	return nil
}

// SignToken signs a jwt token. Much of this is taken from here: https://github.com/google/exposure-notifications-verification-server/blob/main/pkg/jwthelper/jwthelper.go
func SignToken(token *jwt.Token, signer crypto.Signer) (string, error) {
	signingString, err := token.SigningString()
	if err != nil {
		return "", err
	}

	digest := sha256.Sum256([]byte(signingString))

	sig, err := signer.Sign(rand.Reader, digest[:], nil)
	if err != nil {
		return "", fmt.Errorf("error signing token: %w", err)
	}

	// Unpack the ASN1 signature. ECDSA signers are supposed to return this format
	// https://golang.org/pkg/crypto/#Signer
	// All supported signers in thise codebase are verified to return ASN1.
	var parsedSig struct{ R, S *big.Int }
	// ASN1 is not the expected format for an ES256 JWT signature.
	// The output format is specified here, https://tools.ietf.org/html/rfc7518#section-3.4
	// Reproduced here for reference.
	//    The ECDSA P-256 SHA-256 digital signature is generated as follows:
	//
	// 1 .  Generate a digital signature of the JWS Signing Input using ECDSA
	//      P-256 SHA-256 with the desired private key.  The output will be
	//      the pair (R, S), where R and S are 256-bit unsigned integers.
	if _, err := asn1.Unmarshal(sig, &parsedSig); err != nil {
		return "", fmt.Errorf("unable to unmarshal signature: %w", err)
	}

	keyBytes := 256 / 8
	if 256%8 > 0 {
		keyBytes++
	}

	// 2. Turn R and S into octet sequences in big-endian order, with each
	// 		array being be 32 octets long.  The octet sequence
	// 		representations MUST NOT be shortened to omit any leading zero
	// 		octets contained in the values.
	rBytes := parsedSig.R.Bytes()
	rBytesPadded := make([]byte, keyBytes)
	copy(rBytesPadded[keyBytes-len(rBytes):], rBytes)

	sBytes := parsedSig.S.Bytes()
	sBytesPadded := make([]byte, keyBytes)
	copy(sBytesPadded[keyBytes-len(sBytes):], sBytes)

	// 3. Concatenate the two octet sequences in the order R and then S.
	//	 	(Note that many ECDSA implementations will directly produce this
	//	 	concatenation as their output.)
	sig = make([]byte, 0, len(rBytesPadded)+len(sBytesPadded))
	sig = append(sig, rBytesPadded...)
	sig = append(sig, sBytesPadded...)

	return strings.Join([]string{signingString, jwt.EncodeSegment(sig)}, "."), nil
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

// ValidateJWT takes a jwt string, converts it to a JWT, and validates the signature.
func ValidateJWT(keySet jwk.Set, jwtStr string) (jwt2.Token, error) {
	verifiedToken, err := jwt2.Parse([]byte(jwtStr), jwt2.WithKeySet(keySet, jws.WithInferAlgorithmFromKey(true)))
	if err != nil {
		return nil, fmt.Errorf("failed to verify jwt %s: %w", jwtStr, err)
	}

	return verifiedToken, nil
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
