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
	"fmt"
	"net"
	"os/signal"
	"syscall"

	kms "cloud.google.com/go/kms/apiv1"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/pkg/cfgloader"
	"github.com/abcxyz/pkg/grpcutil"
	"github.com/abcxyz/pkg/logging"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	logger := logging.NewFromEnv("")
	ctx = logging.WithLogger(ctx, logger)

	if err := realMain(ctx); err != nil {
		done()
		logger.Fatal(err)
	}
	logger.Infof("successful shutdown")
}

func realMain(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(
		otelgrpc.UnaryServerInterceptor(),
	))

	// TODO(#124): We shouldn't need JVS_ prefix since it's a JVS service.
	cfg := &config.JustificationConfig{}
	if err := cfgloader.Load(ctx, cfg, cfgloader.WithEnvPrefix("JVS_")); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup kms client: %w", err)
	}

	authHandler, err := grpcutil.NewJWTAuthenticationHandler(ctx, grpcutil.NoJWTAuthValidation())
	if err != nil {
		return fmt.Errorf("failed to setup grpc auth handler: %w", err)
	}

	p := justification.NewProcessor(kmsClient, cfg, authHandler)
	jvsAgent := justification.NewJVSAgent(p)
	jvspb.RegisterJVSServiceServer(s, jvsAgent)
	reflection.Register(s)

	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", cfg.Port, err)
	}

	// TODO: Do we need a gRPC health check server?
	// https://github.com/grpc/grpc/blob/master/doc/health-checking.md
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		logger.Debugf("server listening at", zap.Any("address", lis.Addr()))
		return s.Serve(lis)
	})

	// Either we have received a TERM signal or errgroup has encountered an err.
	<-ctx.Done()
	s.GracefulStop()

	if err := g.Wait(); err != nil {
		return fmt.Errorf("error running server: %w", err)
	}
	return nil
}
