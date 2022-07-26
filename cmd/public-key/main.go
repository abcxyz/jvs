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

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/jvs/pkg/util"
	"github.com/abcxyz/pkg/cache"
	"github.com/abcxyz/pkg/logging"
	"go.uber.org/zap"
)

func main() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	logger := logging.NewFromEnv("")
	ctx = logging.WithLogger(ctx, logger)

	if err := realMain(ctx); err != nil {
		done()
		log.Fatal(err)
	}
}

// realMain creates an HTTP server for use with hosting public certs.
// This server supports graceful stopping and cancellation by:
//   - using a cancellable context
//   - listening to incoming requests in a goroutine.
func realMain(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup kms client: %w", err)
	}
	defer util.GracefulClose(logger, kmsClient)

	publicKeyCfg, err := config.LoadPublicKeyConfig(ctx, []byte{})
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cache := cache.New[string](publicKeyCfg.CacheTimeout)

	firestoreDoc, err := util.ParseFirestoreDocResource(publicKeyCfg.FirestoreDocResourceName)
	if err != nil {
		return err
	}

	firestoreClient, err := firestore.NewClient(ctx, firestoreDoc.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to create Firestore client: %w", err)
	}
	defer util.GracefulClose(logger, firestoreClient)

	keyCfg := config.NewFirestoreConfig(firestoreClient, firestoreDoc.DocPath)

	ks := &jvscrypto.KeyServer{
		KMSClient:       kmsClient,
		KeyCfg:          keyCfg,
		PublicKeyConfig: publicKeyCfg,
		Cache:           cache,
	}

	mux := http.NewServeMux()
	mux.Handle("/.well-known/jwks", ks)

	// Create the server and listen in a goroutine.
	logger.Debug("starting server on port", zap.String("port", publicKeyCfg.Port))
	server := &http.Server{
		Addr:              ":" + publicKeyCfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}
	serverErrCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case serverErrCh <- err:
			default:
			}
		}
	}()

	// Wait for shutdown signal or error from the listener.
	select {
	case err := <-serverErrCh:
		return fmt.Errorf("error from server listener: %w", err)
	case <-ctx.Done():
	}

	// Gracefully shut down the server.
	shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}
	return nil
}
