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
	"github.com/abcxyz/jvs/pkg/render"
)

// Server holds the parsed html templates.
type Server struct {
	c *controller.Controller
}

// NewServer creates a new HTTP server implementation that will handle
// rendering the JVS form using a controller.
func NewServer(ctx context.Context, cfg *config.UIServiceConfig) (*Server, error) {
	// Create the renderer
	h, err := render.NewRenderer(ctx, assets.ServerFS(), cfg.DevMode)
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	return &Server{
		c: controller.New(h, cfg.Allowlist),
	}, nil
}

// Routes creates a ServeMux of all of the routes that
// this Router supports.
func (s *Server) Routes() http.Handler {
	staticFS := assets.ServerStaticFS()
	mux := http.NewServeMux()
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	mux.Handle("/popup", s.c.HandlePopup())
	return mux
}
