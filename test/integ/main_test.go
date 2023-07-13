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

// Main entry point for integration tests.
package integ

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/renderer"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/sethvargo/go-retry"
	"google.golang.org/api/iterator"
	grpccodes "google.golang.org/grpc/codes"
	grpcmetadata "google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestJVS(t *testing.T) {
	t.Parallel()
	testutil.SkipIfNotIntegration(t)

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	keyRing := os.Getenv("TEST_JVS_KMS_KEY_RING")
	if keyRing == "" {
		t.Fatal("Key ring must be provided using TEST_JVS_KMS_KEY_RING env variable.")
	}

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		t.Fatalf("failed to setup kms client: %s", err)
	}

	keyRing = strings.Trim(keyRing, "\"")
	primaryKeyVersion := "1"

	keyName := testCreateKey(ctx, t, kmsClient, keyRing, primaryKeyVersion)
	t.Cleanup(func() {
		testCleanUpKey(ctx, t, kmsClient, keyName)
		if err := kmsClient.Close(); err != nil {
			t.Errorf("Clean up of key %s failed: %v", keyName, err)
		}
	})

	cfg := &config.JustificationConfig{
		ProjectID:          os.Getenv("PROJECT_ID"),
		KeyName:            keyName,
		Issuer:             "ci-test",
		PluginDir:          "/var/jvs/plugins",
		SignerCacheTimeout: 1 * time.Nanosecond, // no caching
		DefaultTTL:         15 * time.Minute,
		MaxTTL:             2 * time.Hour,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	authKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ecdsaKey, err := jwk.FromRaw(authKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	keyID := keyName + "/cryptoKeyVersions/" + primaryKeyVersion
	if err := ecdsaKey.Set(jwk.KeyIDKey, keyID); err != nil {
		t.Fatal(err)
	}

	keySet := testKeySetFromKMS(ctx, t, kmsClient, keyName)

	tok := testutil.CreateJWT(t, "test_id", "requestor@example.com")
	validJWT := testutil.SignToken(t, tok, authKey, keyID)
	ctx = grpcmetadata.NewIncomingContext(ctx, grpcmetadata.New(map[string]string{
		"authorization": "Bearer " + validJWT,
	}))

	p := justification.NewProcessor(kmsClient, cfg)
	jvsAgent := justification.NewJVSAgent(p)

	tests := []struct {
		name          string
		request       *jvspb.CreateJustificationRequest
		wantSubject   string
		wantAudiences []string
		wantErr       string
	}{
		{
			name: "happy_path",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "This is a test.",
					},
				},
				Ttl: &durationpb.Duration{
					Seconds: 3600,
				},
			},
			wantSubject:   "requestor@example.com", // subject inherits requestor
			wantAudiences: []string{justification.DefaultAudience},
		},
		{
			name: "custom_subject",
			request: &jvspb.CreateJustificationRequest{
				Subject: "foo@bar.com",
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "This is a test.",
					},
				},
				Ttl: &durationpb.Duration{
					Seconds: 3600,
				},
			},
			wantSubject:   "foo@bar.com",
			wantAudiences: []string{justification.DefaultAudience},
		},
		{
			name: "custom_audiences",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "This is a test.",
					},
				},
				Ttl: &durationpb.Duration{
					Seconds: 3600,
				},
				Audiences: []string{"aud1", "aud2"},
			},
			wantSubject:   "requestor@example.com", // subject inherits requestor
			wantAudiences: []string{"aud1", "aud2"},
		},
		{
			name: "unknown_justification",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "who-knows",
						Value:    "This is a test.",
					},
				},
				Ttl: &durationpb.Duration{
					Seconds: 3600,
				},
			},
			wantErr: "failed to validate request",
		},
		{
			name: "blank_justification",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{
					{
						Category: "explanation",
						Value:    "",
					},
				},
				Ttl: &durationpb.Duration{
					Seconds: 3600,
				},
			},
			wantErr: "failed to validate request",
		},
		{
			name: "no_justification",
			request: &jvspb.CreateJustificationRequest{
				Justifications: []*jvspb.Justification{},
				Ttl: &durationpb.Duration{
					Seconds: 3600,
				},
			},
			wantErr: "failed to validate request",
		},
		{
			name:    "empty_request",
			request: &jvspb.CreateJustificationRequest{},
			wantErr: "failed to validate request",
		},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			now := time.Now().UTC()

			resp, gotErr := jvsAgent.CreateJustification(ctx, tc.request)
			if diff := testutil.DiffErrString(gotErr, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if gotErr != nil {
				return
			}

			// Validate message headers - we have to parse the full envelope for this.
			message, err := jws.Parse([]byte(resp.Token))
			if err != nil {
				t.Fatal(err)
			}
			sigs := message.Signatures()
			if got, want := len(sigs), 1; got != want {
				t.Errorf("expected length %d to be %d: %#v", got, want, sigs)
			} else {
				headers := sigs[0].ProtectedHeaders()
				if got, want := headers.Type(), "JWT"; got != want {
					t.Errorf("typ: expected %q to be %q", got, want)
				}
				if got, want := string(headers.Algorithm()), "ES256"; got != want {
					t.Errorf("alg: expected %q to be %q", got, want)
				}
				if got, want := headers.KeyID(), keyID; got != want {
					t.Errorf("expected %q to be %q", got, want)
				}
			}

			// Parse as a JWT.
			token, err := jwt.Parse([]byte(resp.Token),
				jwt.WithContext(ctx),
				jwt.WithKeySet(keySet, jws.WithInferAlgorithmFromKey(true)),
				jwt.WithAcceptableSkew(5*time.Second),
				jvspb.WithTypedJustifications(),
			)
			if err != nil {
				t.Fatal(err)
			}

			// Validate standard claims.
			if got, want := token.Audience(), tc.wantAudiences; !reflect.DeepEqual(got, want) {
				t.Errorf("aud: expected %q to be %q", got, want)
			}
			if got := token.Expiration(); !got.After(now) {
				t.Errorf("exp: expected %q to be after %q (%q)", got, now, got.Sub(now))
			}
			if got := token.IssuedAt(); got.IsZero() {
				t.Errorf("iat: expected %q to be", got)
			}
			if got, want := token.Issuer(), "ci-test"; got != want {
				t.Errorf("iss: expected %q to be %q", got, want)
			}
			if got, want := len(token.JwtID()), 36; got != want {
				t.Errorf("jti: expected length %d to be %d: %#v", got, want, token.JwtID())
			}
			if got := token.NotBefore(); !got.Before(now) {
				t.Errorf("nbf: expected %q to be after %q (%q)", got, now, got.Sub(now))
			}
			if got, want := token.Subject(), tc.wantSubject; got != want {
				t.Errorf("sub: expected %q to be %q", got, want)
			}

			// Validate custom claims.
			gotRequestor, err := jvspb.GetRequestor(token)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := gotRequestor, "requestor@example.com"; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}

			gotJustifications, err := jvspb.GetJustifications(token)
			if err != nil {
				t.Fatal(err)
			}
			expectedJustifications := tc.request.Justifications
			if diff := cmp.Diff(expectedJustifications, gotJustifications, cmpopts.IgnoreUnexported(jvspb.Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}
		})
	}
}

// Subtests must be run in sequence, and they have waits in between.
// Therefore, they cannot be parallelized, and aren't a good fit for table testing.
//
//nolint:paralleltest
func TestRotator(t *testing.T) {
	t.Parallel()
	testutil.SkipIfNotIntegration(t)

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	kmsClient, keyName := testSetupRotator(ctx, t)

	cfg := &config.CertRotationConfig{
		ProjectID:        os.Getenv("PROJECT_ID"),
		KeyTTL:           7 * time.Second,
		GracePeriod:      2 * time.Second, // rotate after 5 seconds
		PropagationDelay: time.Second,
		DisabledPeriod:   time.Second,
		KeyNames:         []string{keyName},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	r := jvscrypto.NewRotationHandler(ctx, kmsClient, cfg)

	// Validate we have a single enabled key that is primary.
	testValidateKeyVersionState(ctx, t, kmsClient, keyName, 1,
		map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
			1: kmspb.CryptoKeyVersion_ENABLED,
		})

	t.Run("new_key_creation", func(t *testing.T) {
		time.Sleep(5001 * time.Millisecond) // Wait past the next rotation event
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
			return
		}

		// Validate we have created a new key, but haven't set it as primary yet.
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 1,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_ENABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	t.Run("new_key_promotion", func(t *testing.T) {
		time.Sleep(1001 * time.Millisecond) // Wait past the propagation delay.
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}

		// Validate our new key has been set to primary
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_ENABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	t.Run("old_key_disable", func(t *testing.T) {
		time.Sleep(2001 * time.Millisecond) // Wait past the grace period.
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}
		// Validate that our old key has been disabled.
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_DISABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	t.Run("old_key_destroy", func(t *testing.T) {
		time.Sleep(2001 * time.Millisecond) // Wait past the disabled period and next rotation event.
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}

		// Validate that our old key has been scheduled for destruction, and cycle has started again.
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_DESTROY_SCHEDULED,
				2: kmspb.CryptoKeyVersion_ENABLED,
				3: kmspb.CryptoKeyVersion_ENABLED,
			})
	})
}

//nolint:paralleltest // Subtests need to run in sequence. To parallelize this, but we'd need separate keys from the above (more cruft).
func TestRotator_EdgeCases(t *testing.T) {
	t.Parallel()
	testutil.SkipIfNotIntegration(t)

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	kmsClient, keyName := testSetupRotator(ctx, t)

	cfg := &config.CertRotationConfig{
		ProjectID:        os.Getenv("PROJECT_ID"),
		KeyTTL:           99 * time.Hour,
		GracePeriod:      time.Second,
		PropagationDelay: time.Second,
		DisabledPeriod:   time.Second,
		KeyNames:         []string{keyName},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	r := jvscrypto.NewRotationHandler(ctx, kmsClient, cfg)

	time.Sleep(1001 * time.Millisecond) // Wait past the propagation delay.

	t.Run("invalid_primary", func(t *testing.T) {
		// Set primary to a version that doesn't exist
		if err := jvscrypto.SetPrimary(ctx, kmsClient, keyName, keyName+"/cryptoKeyVersions/99"); err != nil {
			t.Fatalf("unable to set primary: %s", err)
		}
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}

		// Validate that we fixed the situation by setting our valid key to primary
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 1,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_ENABLED,
			})
	})

	t.Run("emergent_disable", func(t *testing.T) {
		// Emergently disable our primary.
		testEmergentDisable(ctx, t, kmsClient, keyName, keyName+"/cryptoKeyVersions/1")

		// Validate that the rotator will fix the situation by creating a new version and setting it to primary
		if err := r.RotateKey(ctx, keyName); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}

		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_DISABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	})
}

func TestPublicKeys(t *testing.T) {
	t.Parallel()
	testutil.SkipIfNotIntegration(t)

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		t.Fatalf("failed to setup kms client: %s", err)
	}
	keyRing := os.Getenv("TEST_JVS_KMS_KEY_RING")
	if keyRing == "" {
		t.Fatal("Key ring must be provided using TEST_JVS_KMS_KEY_RING env variable.")
	}
	keyRing = strings.Trim(keyRing, "\"")
	primaryKeyVersion := "1"
	keyName := testCreateKey(ctx, t, kmsClient, keyRing, primaryKeyVersion)

	cfg := &config.PublicKeyConfig{
		ProjectID:    os.Getenv("PROJECT_ID"),
		KeyNames:     []string{keyName},
		CacheTimeout: 10 * time.Second,
	}

	h, err := renderer.New(ctx, nil,
		renderer.WithDebug(cfg.DevMode))
	if err != nil {
		t.Fatal(err)
	}

	keyServer := jvscrypto.NewKeyServer(ctx, kmsClient, cfg, h)

	publicKeys1, publicKeysStr1 := testPublicKeysFromKMS(ctx, t, kmsClient, keyName)

	if got, want := len(publicKeys1), 1; got != want {
		t.Errorf("got %d public keys, expected %d: %#v", got, want, publicKeys1)
	}

	// test for one key version
	testValidatePublicKeys(ctx, t, keyServer, publicKeysStr1)

	testCreateKeyVersion(ctx, t, kmsClient, keyName)
	// test for cache mechanism
	testValidatePublicKeys(ctx, t, keyServer, publicKeysStr1)
	// Wait for the cache timeout
	time.Sleep(10 * time.Second)
	publicKeys2, publicKeysStr2 := testPublicKeysFromKMS(ctx, t, kmsClient, keyName)

	if got, want := len(publicKeys2), 2; got != want {
		t.Errorf("got %d public keys, expected %d: %#v", got, want, publicKeys2)
	}

	// test for cache timeout mechanism and multiple key version
	testValidatePublicKeys(ctx, t, keyServer, publicKeysStr2)
	t.Cleanup(func() {
		testCleanUpKey(ctx, t, kmsClient, keyName)
		if err := kmsClient.Close(); err != nil {
			t.Errorf("Clean up of key %s failed: %v", keyName, err)
		}
	})
}

// These tests must be run in sequence, and they have waits in between.
// Therefore, they cannot be parallelized, and aren't a good fit for table
// testing.
//
//nolint:tparallel
func TestCertActions(t *testing.T) {
	t.Parallel()
	testutil.SkipIfNotIntegration(t)

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	kmsClient, keyName := testSetupRotator(ctx, t)

	cfg := &config.CertRotationConfig{
		ProjectID:        os.Getenv("PROJECT_ID"),
		KeyTTL:           7 * time.Second,
		GracePeriod:      2 * time.Second, // rotate after 5 seconds
		PropagationDelay: time.Second,
		DisabledPeriod:   time.Second,
		KeyNames:         []string{keyName},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}

	r := jvscrypto.NewRotationHandler(ctx, kmsClient, cfg)

	s := &jvscrypto.CertificateActionService{
		Handler:   r,
		KMSClient: kmsClient,
	}

	// Validate we have a single enabled key that is primary.
	testValidateKeyVersionState(ctx, t, kmsClient, keyName, 1,
		map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
			1: kmspb.CryptoKeyVersion_ENABLED,
		})

	if pass := t.Run("graceful_rotation", func(t *testing.T) {
		actions := []*jvspb.Action{
			{
				Version: keyName + "/cryptoKeyVersions/1",
				Action:  jvspb.Action_ROTATE,
			},
		}
		if _, err := s.CertificateAction(ctx, &jvspb.CertificateActionRequest{Actions: actions}); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}

		// Validate we have created a new key, and set it as primary
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_ENABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	}); !pass {
		t.FailNow()
	}

	if pass := t.Run("no_op", func(t *testing.T) {
		actions := []*jvspb.Action{
			{
				Version: keyName + "/cryptoKeyVersions/1",
				Action:  jvspb.Action_ROTATE,
			},
		}
		if _, err := s.CertificateAction(ctx, &jvspb.CertificateActionRequest{Actions: actions}); err != nil {
			t.Fatalf("err when trying to rotate: %s", err)
		}

		// 1 is not a primary, so calling rotate on it again should do nothing.
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 2,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_ENABLED,
				2: kmspb.CryptoKeyVersion_ENABLED,
			})
	}); !pass {
		t.FailNow()
	}

	if pass := t.Run("force_disable", func(t *testing.T) {
		actions := []*jvspb.Action{
			{
				Version: keyName + "/cryptoKeyVersions/1",
				Action:  jvspb.Action_FORCE_DISABLE,
			},
			{
				Version: keyName + "/cryptoKeyVersions/2",
				Action:  jvspb.Action_FORCE_DISABLE,
			},
		}
		if _, err := s.CertificateAction(ctx, &jvspb.CertificateActionRequest{Actions: actions}); err != nil {
			t.Fatalf("err when trying to disable: %s", err)
		}

		// Validate we created a new key, and set it to primary, disabled 2 others.
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 3,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_DISABLED,
				2: kmspb.CryptoKeyVersion_DISABLED,
				3: kmspb.CryptoKeyVersion_ENABLED,
			})
	}); !pass {
		t.FailNow()
	}

	if pass := t.Run("force_destroy", func(t *testing.T) {
		actions := []*jvspb.Action{
			{
				Version: keyName + "/cryptoKeyVersions/2",
				Action:  jvspb.Action_FORCE_DESTROY,
			},
			{
				Version: keyName + "/cryptoKeyVersions/3",
				Action:  jvspb.Action_FORCE_DESTROY,
			},
		}
		if _, err := s.CertificateAction(ctx, &jvspb.CertificateActionRequest{Actions: actions}); err != nil {
			t.Fatalf("err when trying to destroy: %s", err)
		}

		// Validate we created a new key, and scheduled 2&3 for destroying.
		testValidateKeyVersionState(ctx, t, kmsClient, keyName, 4,
			map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
				1: kmspb.CryptoKeyVersion_DISABLED,
				2: kmspb.CryptoKeyVersion_DESTROY_SCHEDULED,
				3: kmspb.CryptoKeyVersion_DESTROY_SCHEDULED,
				4: kmspb.CryptoKeyVersion_ENABLED,
			})
	}); !pass {
		t.FailNow()
	}
}

// Set up KMS, create a key, and set the primary.
func testSetupRotator(ctx context.Context, tb testing.TB) (*kms.KeyManagementClient, string) {
	tb.Helper()
	keyRing := os.Getenv("TEST_JVS_KMS_KEY_RING")
	if keyRing == "" {
		tb.Fatal("Key ring must be provided using TEST_JVS_KMS_KEY_RING env variable.")
	}

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		tb.Fatalf("failed to setup kms client: %s", err)
	}

	keyRing = strings.Trim(keyRing, "\"")
	primaryKeyVersion := "1"
	keyName := testCreateKey(ctx, tb, kmsClient, keyRing, primaryKeyVersion)
	tb.Cleanup(func() {
		testCleanUpKey(ctx, tb, kmsClient, keyName)
		if err := kmsClient.Close(); err != nil {
			tb.Errorf("Clean up of key %s failed: %s", keyName, err)
		}
	})
	if err := jvscrypto.SetPrimary(ctx, kmsClient, keyName, keyName+"/cryptoKeyVersions/1"); err != nil {
		tb.Fatalf("unable to set primary: %s", err)
	}

	// Validate we have a single enabled key that is primary.
	testValidateKeyVersionState(ctx, tb, kmsClient, keyName, 1,
		map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState{
			1: kmspb.CryptoKeyVersion_ENABLED,
		})

	return kmsClient, keyName
}

// This is intended to mock an event where we need to emergently rotate the key.
// We disable the key version and remove it as primary.
func testEmergentDisable(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName, versionName string) {
	tb.Helper()
	ver, err := kmsClient.GetCryptoKeyVersion(ctx, &kmspb.GetCryptoKeyVersionRequest{Name: versionName})
	if err != nil {
		tb.Fatalf("unable to retrieve version %s: %s", versionName, err)
	}

	ver.State = kmspb.CryptoKeyVersion_DISABLED
	var messageType *kmspb.CryptoKeyVersion
	mask, err := fieldmaskpb.New(messageType, "state")
	if err != nil {
		tb.Fatalf("unable to create field mask: %s", err)
	}
	updateReq := &kmspb.UpdateCryptoKeyVersionRequest{
		CryptoKeyVersion: ver,
		UpdateMask:       mask,
	}
	if _, err := kmsClient.UpdateCryptoKeyVersion(ctx, updateReq); err != nil {
		tb.Fatalf("unable to disable version: %s", err)
	}

	key, err := kmsClient.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{Name: keyName})
	if err != nil {
		tb.Fatalf("unable to retrieve key %s: %s", keyName, err)
	}

	labels := make(map[string]string, 0)
	key.Labels = labels

	var mT *kmspb.CryptoKey
	mask, err = fieldmaskpb.New(mT, "labels")
	if err != nil {
		tb.Fatalf("unable to create field mask: %s", err)
	}
	if _, err := kmsClient.UpdateCryptoKey(ctx, &kmspb.UpdateCryptoKeyRequest{CryptoKey: key, UpdateMask: mask}); err != nil {
		tb.Fatalf("unable to set labels: %s", err)
	}
}

func testValidateKeyVersionState(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName string,
	expectedPrimary int, expectedStates map[int]kmspb.CryptoKeyVersion_CryptoKeyVersionState,
) {
	tb.Helper()
	// validate that each version is in the expected state.
	it := kmsClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: keyName,
	})
	count := 0
	for {
		ver, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			tb.Fatalf("err while calling kms to get key versions: %s", err)
		}
		number, err := strconv.Atoi(ver.Name[strings.LastIndex(ver.Name, "/")+1:])
		if err != nil {
			tb.Fatalf("couldn't convert version %s to number: %s", ver.Name, err)
		}
		if ver.State != expectedStates[number] {
			tb.Errorf("version %s was in state %s, but expected %s", ver.Name, ver.State, expectedStates[number])
		}
		count++
	}
	if count != len(expectedStates) {
		tb.Errorf("got %d versions, expected %d", count, len(expectedStates))
	}

	// validate the primary is set correctly.
	resp, err := kmsClient.GetCryptoKey(ctx, &kmspb.GetCryptoKeyRequest{Name: keyName})
	if err != nil {
		tb.Fatalf("err while calling kms: %s", err)
	}
	primaryName := resp.Labels[jvscrypto.PrimaryKey]
	primaryLabel := primaryName[strings.LastIndex(primaryName, "/")+1:]
	primaryNumber, err := strconv.Atoi(strings.TrimPrefix(primaryLabel, jvscrypto.PrimaryLabelPrefix))
	if err != nil {
		tb.Fatalf("couldn't convert version %s to number: %s", primaryName, err)
	}
	if primaryNumber != expectedPrimary {
		tb.Errorf("primary was set to version %d, but expected %d", primaryNumber, expectedPrimary)
	}
}

// Create an asymmetric signing key for use with integration tests.
func testCreateKey(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyRing, primaryKeyVersion string) string {
	tb.Helper()

	keyName := testKeyName(tb)

	// 'Primary' field will be omitted for keys with purpose other than ENCRYPT_DECRYPT(https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys).
	// Therefore, use `Labels` filed to set the primary key version name.
	labels := map[string]string{jvscrypto.PrimaryKey: jvscrypto.PrimaryLabelPrefix + primaryKeyVersion}
	ck, err := kmsClient.CreateCryptoKey(ctx, &kmspb.CreateCryptoKeyRequest{
		Parent:      keyRing,
		CryptoKeyId: keyName,
		CryptoKey: &kmspb.CryptoKey{
			Purpose: kmspb.CryptoKey_ASYMMETRIC_SIGN,
			VersionTemplate: &kmspb.CryptoKeyVersionTemplate{
				Algorithm: kmspb.CryptoKeyVersion_EC_SIGN_P256_SHA256,
			},
			Labels: labels,
		},

		// Do not create the initial version - we will create one below.
		SkipInitialVersionCreation: true,
	})
	if err != nil {
		tb.Fatalf("failed to create crypto key: %s", err)
	}

	testCreateKeyVersion(ctx, tb, kmsClient, ck.Name)
	return ck.Name
}

// Destroy all versions within a key in order to clean up tests.
func testCleanUpKey(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName string) {
	tb.Helper()

	it := kmsClient.ListCryptoKeyVersions(ctx, &kmspb.ListCryptoKeyVersionsRequest{
		Parent: keyName,
	})

	for {
		ver, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			tb.Fatalf("err while reading crypto key version list: %s", err)
		}

		if ver.State == kmspb.CryptoKeyVersion_DESTROYED ||
			ver.State == kmspb.CryptoKeyVersion_DESTROY_SCHEDULED {
			// no need to destroy again
			continue
		}

		if _, err := kmsClient.DestroyCryptoKeyVersion(ctx, &kmspb.DestroyCryptoKeyVersionRequest{
			Name: ver.Name,
		}); err != nil {
			// Cloud KMS returns the following errors when the key is already
			// destroyed or does not exist.
			code := grpcstatus.Code(err)
			if code == grpccodes.NotFound || code == grpccodes.FailedPrecondition {
				return
			}

			tb.Errorf("cleanup: failed to destroy crypto key version %q: %s", ver.Name, err)
		}
	}
}

// Build a key set containing the public key for the first key version from the specified key. Static, does not update automatically.
func testKeySetFromKMS(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName string) jwk.Set {
	tb.Helper()
	keySet := jwk.NewSet()
	versionName := keyName + "/cryptoKeyVersions/1"
	pubKeyResp, err := kmsClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: versionName})
	if err != nil {
		tb.Fatalf("Couldn't retrieve public key: %s", err)
	}
	block, _ := pem.Decode([]byte(pubKeyResp.Pem))
	if block == nil || block.Type != "PUBLIC KEY" {
		tb.Fatal("failed to decode PEM block containing public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		tb.Fatalf("Couldn't parse public key: %s", err)
	}

	jwkKey, err := jwk.FromRaw(pub)
	if err != nil {
		tb.Fatalf("Couldn't convert key to jwk: %s", err)
	}
	if err := jwkKey.Set("kid", versionName); err != nil {
		tb.Fatalf("Couldn't set key id: %s", err)
	}
	if err := keySet.AddKey(jwkKey); err != nil {
		tb.Fatalf("Couldn't add jwk to set: %s", err)
	}
	return keySet
}

// Create a new KeyVersion for use with integration tests.
func testCreateKeyVersion(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName string) string {
	tb.Helper()

	ck, err := kmsClient.CreateCryptoKeyVersion(ctx, &kmspb.CreateCryptoKeyVersionRequest{
		Parent:           keyName,
		CryptoKeyVersion: &kmspb.CryptoKeyVersion{},
	})
	if err != nil {
		tb.Fatalf("failed to create crypto key version: %s", err)
	}

	// Wait for a key version to be created and enabled.
	b := retry.WithMaxRetries(5, retry.NewFibonacci(500*time.Millisecond))
	if err := retry.Do(ctx, b, func(ctx context.Context) error {
		ckv, err := kmsClient.GetCryptoKeyVersion(ctx, &kmspb.GetCryptoKeyVersionRequest{
			Name: ck.Name,
		})
		if err != nil {
			return fmt.Errorf("failed to get crypto key version: %w", err)
		}
		if got, want := ckv.State, kmspb.CryptoKeyVersion_ENABLED; got != want {
			return retry.RetryableError(fmt.Errorf("expected %s to be %s", got.String(), want.String()))
		}
		return nil
	}); err != nil {
		tb.Fatalf("key did not enter ready state: %s", err)
	}
	return ck.Name
}

// Build public keys list and public keys list converted to string(including public keys).
func testPublicKeysFromKMS(ctx context.Context, tb testing.TB, kmsClient *kms.KeyManagementClient, keyName string) (map[string]crypto.PublicKey, string) {
	tb.Helper()

	keyVersions, err := jvscrypto.CryptoKeyVersionsFor(ctx, kmsClient, []string{keyName})
	if err != nil {
		tb.Fatal(err)
	}

	publicKeys, err := jvscrypto.PublicKeysFor(ctx, kmsClient, keyVersions)
	if err != nil {
		tb.Fatal(err)
	}

	jwks, err := jvscrypto.JWKSFromPublicKeys(publicKeys)
	if err != nil {
		tb.Fatal(err)
	}

	b, err := json.Marshal(jwks)
	if err != nil {
		tb.Fatal(err)
	}
	return publicKeys, string(b)
}

func testValidatePublicKeys(ctx context.Context, tb testing.TB, s http.Handler, expectedPublicKeys string) {
	tb.Helper()

	req, err := http.NewRequestWithContext(ctx, "GET", "/.well-known/jwks", nil)
	if err != nil {
		tb.Fatalf("http.NewRequest(): got %v, want no error", err)
	}

	rw := httptest.NewRecorder()
	s.ServeHTTP(rw, req)
	if gotCode, wantCode := rw.Code, http.StatusOK; gotCode != wantCode {
		tb.Errorf("Response Code: got %d, want %d", gotCode, wantCode)
		return
	}
	got := rw.Body.String()

	if diff := cmp.Diff(expectedPublicKeys, got); diff != "" {
		tb.Errorf("GotPublicKeys diff (-want, +got): %v", diff)
	}
}

// testKeyName creates a name with a semi-predicatable name and a random suffix.
func testKeyName(tb testing.TB) string {
	tb.Helper()

	prefix := time.Now().UTC().Format("20060201")

	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		tb.Fatalf("failed to read random bytes: %s", err)
	}

	return prefix + "-" + hex.EncodeToString(b)
}
