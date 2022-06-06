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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/pkg/logging"
	"github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

type server struct {
	handler *jvscrypto.RotationHandler
}

// ServeHTTP rotates a single key's versions.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	logger.Info("received request", zap.Any("url", r.URL))

	var errs error
	for _, key := range s.handler.CryptoConfig.KeyRings {
		if err := s.handler.RotateKey(r.Context(), key); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("error while rotating key ring %s: %w", key, err))
			continue
		}
		logger.Info("successfully performed actions (if necessary) on key.", zap.String("key", key))
	}
	if errs != nil {
		logger.Error("ran into errors while rotating keys", zap.Error(errs))
		http.Error(w, "error while rotating keys", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, "finished with all keys successfully.\n") // automatically calls `w.WriteHeader(http.StatusOK)`
}

func main() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	logger := logging.NewFromEnv("")
	ctx = logging.WithLogger(ctx, logger)

	if err := realMain(ctx); err != nil {
		done()
		logger.Fatal(err)
	}
}

// realMain creates an HTTP server for use with rotating certificates.
// This server supports graceful stopping and cancellation by:
//   - using a cancellable context
//   - listening to incoming requests in a goroutine
func realMain(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup kms client: %w", err)
	}
	defer kmsClient.Close()

	config, err := config.LoadCryptoConfig(ctx, []byte{})
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	handler := &jvscrypto.RotationHandler{
		KMSClient:    kmsClient,
		CryptoConfig: config,
	}

	mux := http.NewServeMux()
	mux.Handle("/", &server{
		handler: handler,
	})

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		logger.Debug("defaulting to port ", zap.String("port", port))
	}

	// Create the server and listen in a goroutine.
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
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
