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
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/abcxyz/jvs/pkg/ui"
	"github.com/abcxyz/pkg/logging"
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

func realMain(ctx context.Context) error {
	cfg, err := ui.NewConfig(ctx)
	if err != nil {
		return fmt.Errorf("server.NewConfig: %w", err)
	}

	tmplLocations := constructTmplMap("./assets/templates")

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

// TODO use filepath.WalkDir to dynamically generate this map.
func constructTmplMap(root string) map[string]string {
	// tmplMap := make(map[string]string)

	// filepath.WalkDir(root, func(path string, di fs.DirEntry, err error) error {
	// 	fmt.Printf("Visited: %s\n", path)
	// 	return nil
	// })

	return map[string]string{
		"popup":     "./assets/templates/popup.html.tmpl",
		"success":   "./assets/templates/success.html.tmpl",
		"forbidden": "./assets/templates/forbidden.html.tmpl",
	}
}
