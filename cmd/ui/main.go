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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/abcxyz/jvs/pkg/cli"
	"github.com/abcxyz/pkg/logging"
)

func main() {
	ctx, done := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer done()

	logger := logging.NewFromEnv("")
	ctx = logging.WithLogger(ctx, logger)

	fmt.Fprintf(os.Stderr, "WARNING! The ui binary is deprecated. Please use "+
		"`jvsctl ui server` instead. This binary will be removed in a future "+
		"release.\n\n")

	args := os.Args[1:]
	args = append([]string{"ui", "server"}, args...)
	if err := cli.Run(ctx, args); err != nil {
		done()
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
