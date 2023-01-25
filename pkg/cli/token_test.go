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
	"reflect"
	"strings"
	"testing"
	"time"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/grpc"
)

func TestNewTokenCmd(t *testing.T) {
	t.Parallel()

	goodJVS, _ := testutil.FakeGRPCServer(t, func(s *grpc.Server) {
		jvspb.RegisterJVSServiceServer(s, &fakeJVS{})
	})
	badJVS, _ := testutil.FakeGRPCServer(t, func(s *grpc.Server) {
		jvspb.RegisterJVSServiceServer(s, &fakeJVS{returnErr: fmt.Errorf("testing server error")})
	})

	cases := []struct {
		name              string
		config            *config.CLIConfig
		args              []string
		expJustifications []*jvspb.Justification
		expErr            string
	}{
		{
			name:   "too_many_args",
			args:   []string{"foo"},
			expErr: `accepts 0 arg(s)`,
		},
		{
			name:   "missing_explanation",
			args:   nil,
			expErr: `"explanation" not set`,
		},
		{
			name: "bad_server_response",
			config: &config.CLIConfig{
				Server:   badJVS,
				Insecure: true,
			},
			args:   []string{"-e", "for testing purposes", "--disable-authn"},
			expErr: "testing server error",
		},
		{
			name: "happy_path",
			config: &config.CLIConfig{
				Server:   goodJVS,
				Insecure: true,
			},
			args: []string{"-e=for testing purposes", "--disable-authn"},
			expJustifications: []*jvspb.Justification{
				{
					Category: "explanation",
					Value:    "for testing purposes",
				},
			},
		},
		{
			name:   "breakglass",
			config: &config.CLIConfig{},
			args:   []string{"-e=prod is down", "--breakglass", "--iat=0"},
			expJustifications: []*jvspb.Justification{
				{
					Category: "breakglass",
					Value:    "prod is down",
				},
			},
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			now := time.Unix(0, 0).UTC()

			cmd := newTokenCmd(tc.config)
			stdout, _, err := testExecuteCommand(t, cmd, tc.args...)
			if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
				t.Fatal(diff)
			}
			if err != nil {
				return
			}

			tokenStr := strings.TrimSpace(stdout)
			token, err := jwt.ParseString(tokenStr,
				jwt.WithVerify(false),
				jwt.WithValidate(false),
				jvspb.WithTypedJustifications())
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
			if got, want := token.Subject(), "jvsctl"; got != want {
				t.Errorf("sub: expected %q to be %q", got, want)
			}

			// Validate custom claims.
			justifications, err := jvspb.GetJustifications(token)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(tc.expJustifications, justifications, cmpopts.IgnoreUnexported(jvspb.Justification{})); diff != "" {
				t.Errorf("justs: diff (-want, +got):\n%s", diff)
			}
		})
	}
}

type fakeJVS struct {
	jvspb.UnimplementedJVSServiceServer
	returnErr error
}

func (j *fakeJVS) CreateJustification(_ context.Context, req *jvspb.CreateJustificationRequest) (*jvspb.CreateJustificationResponse, error) {
	if j.returnErr != nil {
		return nil, j.returnErr
	}

	now := time.Unix(0, 0).UTC()
	token, err := jwt.NewBuilder().
		Audience([]string{justification.DefaultAudience}).
		Expiration(now.Add(5 * time.Minute)).
		IssuedAt(now).
		Issuer(Issuer).
		JwtID("test-jwt").
		NotBefore(now).
		Subject(Subject).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	if err := jvspb.SetJustifications(token, req.Justifications); err != nil {
		return nil, fmt.Errorf("failed to set justifications: %w", err)
	}

	b, err := jwt.Sign(token, jwt.WithKey(jwa.HS256, []byte("testing")))
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return &jvspb.CreateJustificationResponse{
		Token: string(b),
	}, nil
}
