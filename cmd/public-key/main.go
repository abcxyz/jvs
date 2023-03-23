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

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/assets"
	"github.com/abcxyz/jvs/internal/version"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/abcxyz/pkg/cfgloader"
	"github.com/abcxyz/pkg/gcputil"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/renderer"
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
	logger.Debugw("server starting",
		"name", version.Name,
		"commit", version.Commit,
		"version", version.Version)

	projectID := gcputil.ProjectID(ctx)

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup kms client: %w", err)
	}
	defer kmsClient.Close()

	var cfg config.PublicKeyConfig
	if err := cfgloader.Load(ctx, &cfg); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger.Debugw("loaded configuration", "config", cfg)

	// Create the renderer
	h, err := renderer.New(ctx, assets.ServerFS(),
		renderer.WithDebug(cfg.DevMode),
		renderer.WithOnError(func(err error) {
			logger.Errorw("failed to render", "error", err)
		}))
	if err != nil {
		return fmt.Errorf("failed to create renderer: %w", err)
	}

	keyServer := jvscrypto.NewKeyServer(ctx, kmsClient, &cfg, h)

	mux := http.NewServeMux()
	mux.Handle("/.well-known/jwks", logging.HTTPInterceptor(logger, projectID)(keyServer))

	// Create the server and listen in a goroutine.
	logger.Debugw("starting server on port", "port", cfg.Port)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
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
