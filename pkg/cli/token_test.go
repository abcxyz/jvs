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
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/spf13/cobra"
)

type fakeJVS struct {
	jvspb.UnimplementedJVSServiceServer
	returnErr error
}

func (j *fakeJVS) CreateJustification(_ context.Context, req *jvspb.CreateJustificationRequest) (*jvspb.CreateJustificationResponse, error) {
	if j.returnErr != nil {
		return nil, j.returnErr
	}

	if req.Justifications[0].Category != "explanation" {
		return nil, fmt.Errorf("unexpected category: %q", req.Justifications[0].Category)
	}

	return &jvspb.CreateJustificationResponse{
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
			server, _ := testutil.FakeGRPCServer(t, func(s *grpc.Server) { jvspb.RegisterJVSServiceServer(s, tc.jvs) })

			// These are global flags.
			cfg = &config.CLIConfig{
				Server: server,
				Authentication: &config.CLIAuthentication{
					Insecure: true,
				},
			}
			flagTokenExplanation = tc.explanation
			flagTTL = time.Minute
			t.Cleanup(testRunTokenCmdCleanup)

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
	flagBreakglass = true
	flagTokenExplanation = "i-have-reason"
	flagTTL = time.Minute

	// Override timeFunc to have fixed time for test.
	now := time.Now().UTC()
	t.Cleanup(testRunTokenCmdCleanup)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetArgs([]string{"--iat", strconv.FormatInt(now.Unix(), 10)})
	cmd.SetOut(&buf)

	if err := runTokenCmd(cmd, nil); err != nil {
		t.Fatal(err)
	}

	// Validate message headers - we have to parse the full envelope for this.
	message, err := jws.Parse(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if sigs, want := message.Signatures(), 1; len(sigs) != want {
		t.Errorf("expected length %d to be %d: %#v", len(sigs), want, sigs)
	} else {
		headers := sigs[0].ProtectedHeaders()
		if got, want := headers.Type(), "JWT"; got != want {
			t.Errorf("typ: expected %q to be %q", got, want)
		}
		if got, want := string(headers.Algorithm()), "HS256"; got != want {
			t.Errorf("alg: expected %q to be %q", got, want)
		}
	}

	// Parse as a JWT.
	token, err := jwt.ParseInsecure(buf.Bytes(), jvspb.WithTypedJustifications())
	if err != nil {
		t.Fatal(err)
	}

	// Validate standard claims.
	if got, want := token.Audience(), []string{justification.DefaultAudience}; !reflect.DeepEqual(got, want) {
		t.Errorf("aud: expected %q to be %q", got, want)
	}
	if got := token.Expiration(); !got.After(now) {
		t.Errorf("exp: expected %q to be after %q (%q)", got, now, got.Sub(now))
	}
	if got := token.IssuedAt(); got.IsZero() {
		t.Errorf("iat: expected %q to be", got)
	}
	if got, want := token.Issuer(), "jvsctl"; got != want {
		t.Errorf("iss: expected %q to be %q", got, want)
	}
	if got, want := len(token.JwtID()), 36; got != want {
		t.Errorf("jti: expected length %d to be %d: %#v", got, want, token.JwtID())
	}
	if got := token.NotBefore(); !got.Before(now) {
		t.Errorf("nbf: expected %q to be after %q (%q)", got, now, got.Sub(now))
	}
	if got, want := token.Subject(), "jvsctl"; got != want {
		t.Errorf("sub: expected %q to be %q", got, want)
	}

	// Validate custom claims.
	gotJustifications, err := jvspb.GetJustifications(token)
	if err != nil {
		t.Fatal(err)
	}
	expectedJustifications := []*jvspb.Justification{
		{
			Category: "breakglass",
			Value:    "i-have-reason",
		},
	}
	if diff := cmp.Diff(expectedJustifications, gotJustifications, cmpopts.IgnoreUnexported(jvspb.Justification{})); diff != "" {
		t.Errorf("justs: diff (-want, +got):\n%s", diff)
	}
}

func testRunTokenCmdCleanup() {
	flagTokenExplanation = ""
	flagTTL = time.Hour
	flagBreakglass = false
	cfg = nil
}
