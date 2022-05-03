package jvscrypto

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"fmt"
	"math/big"
	"strings"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/golang-jwt/jwt"
	"google.golang.org/api/iterator"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

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
