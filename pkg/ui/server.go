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

package ui

import (
	"context"
	"fmt"
	"net/http"

	"github.com/abcxyz/jvs/assets"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/controller"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/renderer"
)

// Server holds the parsed html templates.
type Server struct {
	c      *controller.Controller
	config *config.UIServiceConfig
}

// NewServer creates a new HTTP server implementation that will handle
// rendering the JVS form using a controller.
func NewServer(ctx context.Context, uiCfg *config.UIServiceConfig, p *justification.Processor) (*Server, error) {
	logger := logging.FromContext(ctx)

	// Create the renderer
	h, err := renderer.New(ctx, assets.ServerFS(),
		renderer.WithDebug(uiCfg.DevMode),
		renderer.WithOnError(func(err error) {
			logger.ErrorContext(ctx, "failed to render", "error", err)
		}))
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	uic, err := controller.New(ctx, h, p, uiCfg.Allowlist)
	if err != nil {
		return nil, fmt.Errorf("failed to create controller: %w", err)
	}

	return &Server{
		c:      uic,
		config: uiCfg,
	}, nil
}

// Routes creates a ServeMux of all of the routes that
// this Router supports.
func (s *Server) Routes(ctx context.Context) http.Handler {
	logger := logging.FromContext(ctx)

	staticFS := assets.ServerStaticFS()
	fileServer := http.FileServer(http.FS(staticFS))

	mux := http.NewServeMux()
	// This is the health checkpoint.
	mux.Handle("/health", s.c.HandleHealth())
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	mux.Handle("/popup", s.c.HandlePopup())

	// Middleware
	root := logging.HTTPInterceptor(logger, s.config.ProjectID)(mux)

	return root
}
