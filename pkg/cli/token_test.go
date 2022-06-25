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
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"

	jvsapis "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/pkg/testutil"
	"github.com/golang-jwt/jwt"
	"github.com/google/go-cmp/cmp"
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

func TestRunTokenCmd_WithJVSServer(t *testing.T) {
	// Cannot parallel because the global CLI config.
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
			server, _ := testutil.FakeGRPCServer(t, func(s *grpc.Server) { jvsapis.RegisterJVSServiceServer(s, tc.jvs) })

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

func TestRunTokenCmd_Breakglass(t *testing.T) {
	// Cannot parallel because the global CLI config.
	// These are global flags.
	cfg = &config.CLIConfig{
		Server: "example.com",
		Authentication: &config.CLIAuthentication{
			Insecure: true,
		},
	}
	breakglass = true
	tokenExplanation = "i-have-reason"
	ttl = time.Minute

	// Override timeFunc to have fixed time for test.
	now := time.Now()
	timeFunc = func() time.Time {
		return now
	}
	t.Cleanup(func() {
		timeFunc = time.Now
	})

	buf := &strings.Builder{}
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	if err := runTokenCmd(cmd, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	p := &jwt.Parser{}
	gotClaims := jwt.MapClaims{}
	if _, _, err := p.ParseUnverified(buf.String(), gotClaims); err != nil {
		t.Errorf("unable to parse token got: %v", err)
	}
	// Don't compare jti since it's randomly generated
	delete(gotClaims, "jti")

	wantClaims := jwt.MapClaims{
		"aud": "TODO #22",
		"iss": "jvsctl",
		"sub": "jvsctl",
		"iat": float64(now.UTC().Unix()),
		"exp": float64(now.UTC().Add(ttl).Unix()),
		"nbf": float64(now.UTC().Unix()),
		"justs": []interface{}{map[string]interface{}{
			"category": "breakglass",
			"value":    "i-have-reason",
		}},
	}

	if diff := cmp.Diff(wantClaims, gotClaims); diff != "" {
		t.Errorf("breakglass token claims (-want,+got):\n%s", diff)
	}
}
