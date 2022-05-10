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
	"os"
	"os/signal"
	"syscall"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/jvscrypto"
	"github.com/hashicorp/go-multierror"
)

type server struct {
	handler *jvscrypto.RotationHandler
}

// HTTPMessage is the request format we will send from cloud scheduler.
type HTTPMessage struct {
	// TODO: We should support manual actions through call arguments, such as a rotation before the TTL. https://github.com/abcxyz/jvs/issues/9
}

// ServeHTTP rotates a single key's versions.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("received request at %s\n", r.URL)

	var errs error
	// TODO: load keys from DB instead. https://github.com/abcxyz/jvs/issues/17
	for _, key := range s.handler.CryptoConfig.KeyNames {
		if err := s.handler.RotateKey(r.Context(), key); err != nil {
			errs = multierror.Append(errs, fmt.Errorf("error while rotating key %s : %v\n", key, err))
			continue
		}
		log.Printf("successfully performed actions (if necessary) on key: %s.\n", key)
	}
	if errs != nil {
		log.Printf("ran into errors while rotating keys. %v\n", errs)
		http.Error(w, "error while rotating keys", http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, "finished with all keys successfully.\n") // automatically calls `w.WriteHeader(http.StatusOK)`
}

func main() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	if err := realMain(ctx); err != nil {
		done()
		log.Fatal(err)
	}
}

// realMain creates an HTTP server for use with rotating certificates.
// This server supports graceful stopping and cancellation by:
//   - using a cancellable context
//   - listening to incoming requests in a goroutine
func realMain(ctx context.Context) error {
	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup kms client: %v", err)
	}
	defer kmsClient.Close()

	config, err := config.LoadCryptoConfig(ctx, []byte{})
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	handler := &jvscrypto.RotationHandler{
		KmsClient:    kmsClient,
		StateStore:   &jvscrypto.KeyLabelStateStore{KmsClient: kmsClient},
		CryptoConfig: config,
		CurrentTime:  time.Now(),
	}

	mux := http.NewServeMux()
	mux.Handle("/", &server{
		handler: handler,
	})

	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("defaulting to port %s", port)
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
