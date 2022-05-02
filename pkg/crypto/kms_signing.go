package crypto

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"strings"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/golang-jwt/jwt"
	"google.golang.org/api/iterator"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

type KMSSigner struct {
	crypto.Signer // https://golang.org/pkg/crypto/#Signer
	Config        *config.JustificationConfig
	KMSClient     *kms.KeyManagementClient
}

func (k *KMSSigner) Sign(_ io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	ctx := context.Background()

	ver, err := k.getLatestKeyVersion(ctx)
	if err != nil {
		return nil, err
	}

	response, err := k.KMSClient.AsymmetricSign(ctx, &kmspb.AsymmetricSignRequest{
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{
				Sha256: digest,
			},
		},
		Name: ver.Name,
	})
	if err != nil {
		fmt.Printf("Error signing with kms client %v", err)
		return nil, err
	}
	return response.Signature, nil
}

func (k *KMSSigner) Public() crypto.PublicKey {
	ctx := context.Background()
	ver, err := k.getLatestKeyVersion(ctx)
	if err != nil {
		fmt.Printf("Error getting key version %v", err)
		return nil
	}

	response, err := k.KMSClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: ver.Name})
	if err != nil {
		fmt.Printf("Error getting GetPublicKey %v", err)
		return nil
	}
	pubKeyBlock, _ := pem.Decode([]byte(response.Pem))

	pub, err := x509.ParsePKIXPublicKey(pubKeyBlock.Bytes)
	if err != nil {
		fmt.Printf("Error parsing PublicKey %v", err)
		return nil
	}
	return pub.(*rsa.PublicKey)
}

// Look up the newest enabled key version
func (k *KMSSigner) getLatestKeyVersion(ctx context.Context) (*kmspb.CryptoKeyVersion, error) {
	it := k.KMSClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: k.Config.KeyName,
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
		if ver.State == kmspb.CryptoKeyVersion_ENABLED && (newestEnabledVersion == nil || ver.CreateTime.AsTime().After(newestTime)) {
			newestEnabledVersion = ver
			newestTime = ver.CreateTime.AsTime()
		}
	}
	return newestEnabledVersion, nil
}

func (k *KMSSigner) PublicKey(ctx context.Context) ([]byte, error) {
	ver, err := k.getLatestKeyVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %v", err)
	}
	// Retrieve the public key from KMS.
	response, err := k.KMSClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: ver.Name})
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %v", err)
	}
	return []byte(response.Pem), nil
}

func (k *KMSSigner) VerifyJWTString(ctx context.Context, jwtStr string) error {
	key, err := k.PublicKey(ctx)
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
