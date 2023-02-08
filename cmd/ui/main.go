// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"html/template"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/abcxyz/jvs/pkg/ui"
	"github.com/abcxyz/pkg/logging"
)

type Template struct {
	Popup   *template.Template
	Success *template.Template
}

var Templates *Template

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

// realMain creates an HTTP server that renders a justification form and handles its submission.
func realMain(ctx context.Context) error {
	cfg, err := ui.NewConfig(ctx)
	if err != nil {
		return fmt.Errorf("server.NewConfig: %w", err)
	}

	tmplLocations := map[string]string{
		"popup":   "./assets/templates/index.html.tmpl",
		"success": "./assets/templates/success.html.tmpl",
	}

	uiServer, err := ui.NewServer(ctx, cfg, tmplLocations)
	if err != nil {
		return fmt.Errorf("server.NewServer: %w", err)
	}

	// Create the server and listen in a goroutine.
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      uiServer.Routes(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  15 * time.Second,
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
