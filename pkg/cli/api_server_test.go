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
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/testutil"
)

func TestAPIServerCommand(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	cases := []struct {
		name   string
		args   []string
		env    map[string]string
		expErr string
	}{
		{
			name:   "too_many_args",
			args:   []string{"foo"},
			expErr: `unexpected arguments: ["foo"]`,
		},
		{
			name: "invalid_config",
			env: map[string]string{
				"JVS_API_SIGNER_CACHE_TIMEOUT": "-5m",
			},
			expErr: `must be a positive duration`,
		},
		{
			name: "starts",
			env: map[string]string{
				"PROJECT_ID": "example-project",
				"JVS_KEY":    "projects/[JVS_PROJECT]/locations/global/keyRings/[JVS_KEYRING]/cryptoKeys/[JVS_KEY]",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, done := context.WithCancel(ctx)
			defer done()

			var cmd APIServerCommand
			cmd.SetLookupEnv(cli.MultiLookuper(
				cli.MapLookuper(tc.env),
				cli.MapLookuper(map[string]string{
					// Make the test choose a random port.
					"PORT": "0",
				}),
			))
			cmd.testKMSClientOptions = []option.ClientOption{
				// Disable auth lookup in these tests, since we don't actually call KMS.
				option.WithoutAuthentication(),
			}
			_, _, _ = cmd.Pipe()

			srv, grpcServer, closer, err := cmd.RunUnstarted(ctx, tc.args)
			defer func() {
				if err := closer.Close(); err != nil {
					t.Error(err)
				}
			}()
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if err != nil {
				return
			}

			serverCtx, serverDone := context.WithCancel(ctx)
			defer serverDone()
			go func() {
				if err := srv.StartGRPC(serverCtx, grpcServer); err != nil {
					t.Error(err)
				}
			}()

			conn, err := grpc.Dial(srv.Addr(),
				grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()

			hcClient := healthpb.NewHealthClient(conn)
			req := new(healthpb.HealthCheckRequest)
			res, err := hcClient.Check(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := res.GetStatus(), healthpb.HealthCheckResponse_SERVING; got != want {
				t.Errorf("expected status %v to be %v", got, want)
			}
		})
	}
}
