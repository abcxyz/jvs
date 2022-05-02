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
	"log"
	"net"
	"os/signal"
	"syscall"

	kms "cloud.google.com/go/kms/apiv1"
	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	jvs_crypto "github.com/abcxyz/jvs/pkg/jvs-crypto"
	"github.com/sethvargo/go-gcpkms/pkg/gcpkms"
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

	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup kms client: %w", err)
	}

	// TODO: We should have a way of asynchronously updating.
	ver, err := jvs_crypto.GetLatestKeyVersion(ctx, kmsClient, cfg.KeyName)
	if err != nil {
		log.Fatalf("failed to get key version: %v", err)
	}
	signer, err := gcpkms.NewSigner(ctx, kmsClient, ver.Name)
	if err != nil {
		log.Fatalf("failed to crate signer: %v", err)
	}

	p := &justification.Processor{
		Signer: signer,
	}
	jvsAgent := justification.NewJVSAgent(p)
	jvspb.RegisterJVSServiceServer(s, jvsAgent)
	reflection.Register(s)

	lis, err := net.Listen("tcp", ":"+cfg.Port)
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
