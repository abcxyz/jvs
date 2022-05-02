package jvs_crypto

import (
	"context"
	"fmt"
	"strings"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/golang-jwt/jwt"
	"google.golang.org/api/iterator"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

// GetLatestKeyVersion looks up the newest enabled key version
func GetLatestKeyVersion(ctx context.Context, kms *kms.KeyManagementClient, keyName string) (*kmspb.CryptoKeyVersion, error) {
	it := kms.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: keyName,
		Filter: "state=ENABLED",
	})

	var newestEnabledVersion *kmspb.CryptoKeyVersion
	var newestTime time.Time
	for {
		ver, err := it.Next()
		if err == iterator.Done {
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
		return nil, fmt.Errorf("failed to get public key: %v", err)
	}
	// Retrieve the public key from KMS.
	response, err := kms.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: ver.Name})
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %v", err)
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
	if err := jwt.SigningMethodES256.Verify(strings.Join(parts[0:2], "."), parts[2], parsedKey); err != nil {
		return fmt.Errorf("unable to verify signed jwt string. %w", err)
	}
	return nil
}
