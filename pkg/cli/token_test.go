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

package cli

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	jvsapis "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/pkg/testutil"
	"github.com/spf13/cobra"
)

type fakeJVS struct {
	jvsapis.UnimplementedJVSServiceServer
	returnErr error
}

func (j *fakeJVS) CreateJustification(_ context.Context, req *jvsapis.CreateJustificationRequest) (*jvsapis.CreateJustificationResponse, error) {
	if j.returnErr != nil {
		return nil, j.returnErr
	}

	if req.Justifications[0].Category != "explanation" {
		return nil, fmt.Errorf("unexpected category: %q", req.Justifications[0].Category)
	}

	return &jvsapis.CreateJustificationResponse{
		Token: fmt.Sprintf("tokenized(%s);ttl=%v", req.Justifications[0].Value, req.Ttl.AsDuration()),
	}, nil
}

func TestRunTokenCmd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		jvs         *fakeJVS
		explanation string
		wantToken   string
		wantErr     string
	}{{
		name:        "success",
		jvs:         &fakeJVS{},
		explanation: "i-have-reason",
		wantToken:   fmt.Sprintf("tokenized(i-have-reason);ttl=%v", time.Minute),
	}, {
		name:        "error",
		jvs:         &fakeJVS{returnErr: fmt.Errorf("server err")},
		explanation: "i-have-reason",
		wantErr:     "server err",
	}}

	for _, tc := range tests {
		// Cannot parallel because the global CLI config.
		t.Run(tc.name, func(t *testing.T) {
			server, _ := testFakeGRPCServer(t, func(s *grpc.Server) { jvsapis.RegisterJVSServiceServer(s, tc.jvs) })

			// These are global flags.
			cfg = &config.CLIConfig{
				Server: server,
				Authentication: &config.CLIAuthentication{
					Insecure: true,
				},
			}
			tokenExplanation = tc.explanation
			ttl = time.Minute

			buf := &strings.Builder{}
			cmd := &cobra.Command{}
			cmd.SetOut(buf)

			err := runTokenCmd(cmd, nil)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected err: %s", diff)
			}

			if gotToken := buf.String(); gotToken != tc.wantToken {
				t.Errorf("justification token got=%q, want=%q", gotToken, tc.wantToken)
			}
		})
	}
}

// Copied over from Lumberjack. TODO: share it in pkg.
func testFakeGRPCServer(tb testing.TB, registerFunc func(*grpc.Server)) (string, *grpc.ClientConn) {
	tb.Helper()

	s := grpc.NewServer()
	tb.Cleanup(func() { s.GracefulStop() })

	registerFunc(s)

	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		tb.Fatalf("net.Listen(tcp, localhost:0) failed: %v", err)
	}

	go func() {
		if err := s.Serve(lis); err != nil {
			tb.Logf("net.Listen(tcp, localhost:0) serve failed: %v", err)
		}
	}()

	addr := lis.Addr().String()
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		tb.Fatalf("failed to dail %q: %s", addr, err)
	}
	return addr, conn
}
