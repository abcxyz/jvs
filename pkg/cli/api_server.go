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

	kms "cloud.google.com/go/kms/apiv1"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/internal/version"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/serving"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var _ cli.Command = (*APIServerCommand)(nil)

type APIServerCommand struct {
	cli.BaseCommand

	cfg *config.JustificationConfig

	// testFlagSetOpts is only used for testing.
	testFlagSetOpts []cli.Option

	// testKMSClientOptions are KMS client options to override during testing.
	testKMSClientOptions []option.ClientOption
}

func (c *APIServerCommand) Desc() string {
	return `Start an API server`
}

func (c *APIServerCommand) Help() string {
	return `
Usage: {{ COMMAND }} [options]

  Start a JVS API server.
`
}

func (c *APIServerCommand) Flags() *cli.FlagSet {
	c.cfg = &config.JustificationConfig{}
	return c.cfg.ToFlags(c.testFlagSetOpts...)
}

func (c *APIServerCommand) Run(ctx context.Context, args []string) error {
	server, grpcServer, closer, err := c.RunUnstarted(ctx, args)
	if err != nil {
		return err
	}
	defer closer()

	return server.StartGRPC(ctx, grpcServer)
}

func (c *APIServerCommand) RunUnstarted(ctx context.Context, args []string) (*serving.Server, *grpc.Server, func(), error) {
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

	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		logging.GRPCUnaryInterceptor(logger, c.cfg.ProjectID),
		otelgrpc.UnaryServerInterceptor(),
	))

	// Create basic health check
	hs := health.NewServer()
	hs.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, hs)

	p := justification.NewProcessor(kmsClient, c.cfg)
	jvsAgent := justification.NewJVSAgent(p)
	jvspb.RegisterJVSServiceServer(grpcServer, jvsAgent)
	reflection.Register(grpcServer)

	server, err := serving.New(c.cfg.Port)
	if err != nil {
		return nil, nil, closer, fmt.Errorf("failed to create serving infrastructure: %w", err)
	}
	return server, grpcServer, closer, nil
}
