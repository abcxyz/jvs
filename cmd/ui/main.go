package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/abcxyz/jvs/pkg/ui"
	"github.com/abcxyz/pkg/logging"
)

func main() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer done()

	logger := logging.NewFromEnv("")
	ctx = logging.WithLogger(ctx, logger)

	ui.RunServer(ctx)
}
