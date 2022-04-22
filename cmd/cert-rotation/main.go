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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/crypto"
)

type server struct {
	handler *crypto.RotationHandler
}

// HTTPMessage is the request format we will send from cloud scheduler.
type HTTPMessage struct {
	Message struct {
		// TODO: We should support manual actions through call arguments, such as a rotation before the TTL. https://github.com/abcxyz/jvs/issues/9
		KeyName string `json:"key_name"`
	} `json:"message"`
}

// ServeHTTP rotates a single key's versions.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request at %s\n", r.URL)

	// TODO: Use LimitReader. https://github.com/abcxyz/jvs/issues/7
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll: %v\n", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var msg HTTPMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		log.Printf("json.Unmarshal: %v\n", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if err := s.handler.RotateKey(r.Context(), msg.Message.KeyName); err != nil {
		log.Printf("error while rotating key versions: %v\n", err)
		http.Error(w, "Error while rotating key versions", http.StatusInternalServerError)
		return
	}

	success := fmt.Sprintf("Successfully rotated key %s.\n", msg.Message.KeyName)
	log.Print(success)
	fmt.Fprint(w, success) // automatically calls `w.WriteHeader(http.StatusOK)`
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
		log.Fatalf("failed to setup kms client: %v", err)
	}

	config, err := config.LoadConfig(ctx, []byte{})
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	defer kmsClient.Close()
	handler := &crypto.RotationHandler{
		KmsClient:    kmsClient,
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
