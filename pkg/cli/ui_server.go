// Copyright 2023 Google LLC
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

package cli

import (
	"context"
	"fmt"
	"net/http"

	kms "cloud.google.com/go/kms/apiv1"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/internal/version"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/jvs/pkg/ui"
	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/serving"
	"google.golang.org/api/option"
)

var _ cli.Command = (*UIServerCommand)(nil)

type UIServerCommand struct {
	cli.BaseCommand

	cfg *config.UIServiceConfig

	// testFlagSetOpts is only used for testing.
	testFlagSetOpts []cli.Option

	// testKMSClientOptions are KMS client options to override during testing.
	testKMSClientOptions []option.ClientOption
}

func (c *UIServerCommand) Desc() string {
	return `Start a UI server for the JVS`
}

func (c *UIServerCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Start a UI server for the JVS.
`
}

func (c *UIServerCommand) Flags() *cli.FlagSet {
	c.cfg = &config.UIServiceConfig{}
	set := cli.NewFlagSet(c.testFlagSetOpts...)
	return c.cfg.ToFlags(set)
}

func (c *UIServerCommand) Run(ctx context.Context, args []string) error {
	server, mux, closer, err := c.RunUnstarted(ctx, args)
	if err != nil {
		return err
	}
	defer closer()

	return server.StartHTTPHandler(ctx, mux)
}

func (c *UIServerCommand) RunUnstarted(ctx context.Context, args []string) (*serving.Server, http.Handler, func(), error) {
	closer := func() {}

	f := c.Flags()
	if err := f.Parse(args); err != nil {
		return nil, nil, closer, fmt.Errorf("failed to parse flags: %w", err)
	}
	args = f.Args()
	if len(args) > 0 {
		return nil, nil, closer, fmt.Errorf("unexpected arguments: %q", args)
	}

	logger := logging.FromContext(ctx)
	logger.Debugw("server starting",
		"name", version.Name,
		"commit", version.Commit,
		"version", version.Version)

	if err := c.cfg.Validate(); err != nil {
		return nil, nil, closer, fmt.Errorf("invalid configuration: %w", err)
	}
	logger.Debugw("loaded configuration", "config", c.cfg)

	kmsClient, err := kms.NewKeyManagementClient(ctx, c.testKMSClientOptions...)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to setup kms client: %w", err)
	}
	closer = func() {
		if err := kmsClient.Close(); err != nil {
			logger.Errorw("failed to close kms client", "error", err)
		}
	}

	plugins, err := jvspb.InitPlugins(ctx, c.cfg.Plugins)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to init plugins: %w", err)
	}

	p := justification.NewProcessor(kmsClient, c.cfg.JustificationConfig, justification.WithPlugins(plugins))

	uiServer, err := ui.NewServer(ctx, c.cfg, p)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create ui server: %w", err)
	}
	mux := uiServer.Routes(ctx)

	server, err := serving.New(c.cfg.Port)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create serving infrastructure: %w", err)
	}

	return server, mux, closer, nil
}
