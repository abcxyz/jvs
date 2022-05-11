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
	"github.com/lestrrat-go/jwx/jwk"
)

type server struct {
	ks *jvscrypto.KeyServer
}

// ServeHTTP rotates a single key's versions.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("received request at %s\n", r.URL)

	jwks := make([]*jwk.Key, 0)
	for _, key := range s.ks.CryptoConfig.KeyNames {
		list, err := s.ks.JWKList(r.Context(), key)
		if err != nil {
			log.Printf("ran into error while determining public keys. %v\n", err)
			http.Error(w, "error determining public keys", http.StatusInternalServerError)
			return
		}
		jwks = append(jwks, list...)
	}
	json, err := jvscrypto.FormatJWKString(jwks)
	if err != nil {
		log.Printf("ran into error while formatting public keys. %v\n", err)
		http.Error(w, "error formatting public keys", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, json)
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

	ks := &jvscrypto.KeyServer{
		KmsClient:    kmsClient,
		CryptoConfig: config,
		StateStore:   &jvscrypto.KeyLabelStateStore{KMSClient: kmsClient},
	}

	mux := http.NewServeMux()
	mux.Handle("/.well-known/jwks", &server{
		ks: ks,
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
