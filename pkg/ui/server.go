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

	kms "cloud.google.com/go/kms/apiv1"
	"github.com/abcxyz/jvs/assets"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/controller"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/jvs/pkg/render"
	"github.com/abcxyz/pkg/cfgloader"
)

// Server holds the parsed html templates.
type Server struct {
	c *controller.Controller
}

// NewServer creates a new HTTP server implementation that will handle
// rendering the JVS form using a controller.
func NewServer(ctx context.Context, uiCfg *config.UIServiceConfig) (*Server, error) {
	// Create the renderer
	h, err := render.NewRenderer(ctx, assets.ServerFS(), uiCfg.DevMode)
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to setup kms client: %w", err)
	}

	var cfg config.JustificationConfig
	if err := cfgloader.Load(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	p := justification.NewProcessor(kmsClient, &cfg)

	return &Server{
		c: controller.New(h, p, uiCfg.Allowlist),
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
