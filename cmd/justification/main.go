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

package justification

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/signal"
	"syscall"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	if err := realMain(ctx); err != nil {
		done()
		log.Fatal(err)
	}
	log.Printf("successful shutdown")
}

func realMain(ctx context.Context) error {
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(
		otelgrpc.UnaryServerInterceptor(),
	))

	cfg, err := config.LoadJustificationConfig(ctx, []byte{})
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	p := &justification.Processor{
		Config: cfg,
	}
	jvsAgent := justification.NewJVSAgent(p)
	jvspb.RegisterJVSServiceServer(s, jvsAgent)
	reflection.Register(s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", cfg.Port, err)
	}

	// TODO: Do we need a gRPC health check server?
	// https://github.com/grpc/grpc/blob/master/doc/health-checking.md
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		log.Printf("server listening at %v", lis.Addr())
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
