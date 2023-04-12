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
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/abcxyz/pkg/cli"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/testutil"
	"github.com/sethvargo/go-envconfig"
	"google.golang.org/api/option"
)

func TestRotationServerCommand(t *testing.T) {
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
				"PROPAGATION_DELAY": "10m",
				"GRACE_PERIOD":      "5m",
			},
			expErr: `must be less than grace period`,
		},
		{
			name: "starts",
			env: map[string]string{
				"PROJECT_ID": "example-project",
				"KEY_NAMES":  "fake/key",
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx, done := context.WithCancel(ctx)
			defer done()

			var cmd RotationServerCommand
			cmd.testFlagSetOpts = []cli.Option{cli.WithLookupEnv(envconfig.MultiLookuper(
				envconfig.MapLookuper(tc.env),
				envconfig.MapLookuper(map[string]string{
					// Make the test choose a random port.
					"PORT": "0",
				}),
			).Lookup)}
			cmd.testKMSClientOptions = []option.ClientOption{
				// Disable auth lookup in these tests, since we don't actually call KMS.
				option.WithoutAuthentication(),
			}
			_, _, _ = cmd.Pipe()

			srv, mux, closer, err := cmd.RunUnstarted(ctx, tc.args)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			defer closer()
			if err != nil {
				return
			}

			serverCtx, serverDone := context.WithCancel(ctx)
			defer serverDone()
			go func() {
				if err := srv.StartHTTPHandler(serverCtx, mux); err != nil {
					t.Error(err)
				}
			}()

			client := &http.Client{
				Timeout: 5 * time.Second,
			}

			uri := "http://" + srv.Addr() + "/healthz"
			req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			if got, want := resp.StatusCode, http.StatusOK; got != want {
				b, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatal(err)
				}
				t.Errorf("expected status code %d to be %d: %s", got, want, string(b))
			}
		})
	}
}
