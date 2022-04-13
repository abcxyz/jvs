// Copyright 2022 Lumberjack authors (see AUTHORS file)
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
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"google-on-gcp/jvs/services/go/pkg/crypto"
)

func main() {
	ctx := context.Background()

	project := os.Getenv("PROJECT_ID")
	if project == "" {
		log.Fatal("You must set PROJECT_ID env variable.")
	}
	log.Printf("Project: %s", project)

	topic := os.Getenv("TOPIC_ID")
	if topic == "" {
		log.Fatal("You must set TOPIC_ID env variable.")
	}
	log.Printf("Topic: %s", topic)

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		log.Fatalf("failed to setup client: %v", err)
	}
	defer kmsClient.Close()

	handler := crypto.KmsHandler{
		KmsClient: kmsClient,
	}

	http.HandleFunc("/", handler.ConsumeMessage)
	// Determine port for HTTP service.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}
	// Start HTTP server.
	srv := &http.Server{Addr: ":" + port}

	serverErrCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			select {
			case serverErrCh <- err:
			default:
			}
		}
	}()

	// Wait for shutdown signal or error from the listener.
	select {
	case err := <-serverErrCh:
		fmt.Errorf("error from server listener: %w", err)
	case <-ctx.Done():
	}

	// Gracefully shut down the server.
	shutdownCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
	defer done()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Errorf("failed to shutdown server: %w", err)
	}
}

