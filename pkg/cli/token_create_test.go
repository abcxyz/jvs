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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/grpc"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/pkg/logging"
	"github.com/abcxyz/pkg/testutil"
)

func TestTokenCreateCommand(t *testing.T) {
	t.Parallel()

	ctx := logging.WithLogger(context.Background(), logging.TestLogger(t))

	now := time.Unix(0, 0).UTC()

	goodJVS, _ := testutil.FakeGRPCServer(t, func(s *grpc.Server) {
		jvspb.RegisterJVSServiceServer(s, &fakeJVS{})
	})
	badJVS, _ := testutil.FakeGRPCServer(t, func(s *grpc.Server) {
		jvspb.RegisterJVSServiceServer(s, &fakeJVS{returnErr: fmt.Errorf("testing server error")})
	})

	cases := []struct {
		name              string
		args              []string
		expSubject        string
		expAudiences      []string
		expJustifications []*jvspb.Justification
		expErr            string
	}{
		{
			name:   "too_many_args",
			args:   []string{"foo"},
			expErr: `unexpected arguments: ["foo"]`,
		},
		{
			name:   "missing_justification",
			args:   nil,
			expErr: `justification is required`,
		},
		{
			name: "bad_server_response",
			args: []string{
				"-justification", "for testing purposes",
				"-server", badJVS,
			},
			expErr: "testing server error",
		},
		{
			name: "explanation_flag_back_compatibility", // TODO(#308): remove this later
			args: []string{
				"-explanation", "for testing purposes",
				"-server", goodJVS,
			},
			expAudiences: []string{justification.DefaultAudience},
			expJustifications: []*jvspb.Justification{
				{
					Category: "explanation",
					Value:    "for testing purposes",
				},
			},
		},
		{
			name: "happy_path",
			args: []string{
				"-justification", "for testing purposes",
				"-server", goodJVS,
			},
			expAudiences: []string{justification.DefaultAudience},
			expJustifications: []*jvspb.Justification{
				{
					Category: "explanation",
					Value:    "for testing purposes",
				},
			},
		},
		{
			name: "happy_path_custom_category",
			args: []string{
				"-category", "jira",
				"-justification", "JIRACOMPONENT/123",
				"-server", goodJVS,
			},
			expAudiences: []string{justification.DefaultAudience},
			expJustifications: []*jvspb.Justification{
				{
					Category: "jira",
					Value:    "JIRACOMPONENT/123",
				},
			},
		},
		{
			name: "breakglass",
			args: []string{
				"-justification=prod is down",
				"-breakglass",
			},
			expAudiences: []string{justification.DefaultAudience},
			expJustifications: []*jvspb.Justification{
				{
					Category: "breakglass",
					Value:    "prod is down",
				},
			},
		},
		{
			name: "custom_subject",
			args: []string{
				"-justification=prod is down",
				"-subject=user@example.com",
				"-breakglass",
			},
			expSubject:   "user@example.com",
			expAudiences: []string{justification.DefaultAudience},
			expJustifications: []*jvspb.Justification{
				{
					Category: "breakglass",
					Value:    "prod is down",
				},
			},
		},
		{
			name: "custom_audiences",
			args: []string{
				"-justification=prod is down",
				"-breakglass",
				"-audience=foo,bar",
				"-audience=baz",
			},
			expAudiences: []string{"foo", "bar", "baz"},
			expJustifications: []*jvspb.Justification{
				{
					Category: "breakglass",
					Value:    "prod is down",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var cmd TokenCreateCommand
			_, stdout, _ := cmd.Pipe()

			args := append([]string{
				// Always append insecure for tests.
				"-insecure",

				// Override timestamp in tests.
				"-now", strconv.FormatInt(now.Unix(), 10),
			}, tc.args...)

			if err := cmd.Run(ctx, args); err != nil {
				if diff := testutil.DiffErrString(err, tc.expErr); diff != "" {
					t.Fatal(diff)
				}
				if err != nil {
					return
				}
			}

			tokenStr := strings.TrimSpace(stdout.String())
			token, err := jwt.ParseInsecure([]byte(tokenStr),
				jwt.WithContext(ctx),
				jwt.WithAcceptableSkew(5*time.Second),
				jvspb.WithTypedJustifications(),
			)
			if err != nil {
				t.Fatal(err)
			}

			// Validate standard claims.
			if got, want := token.Audience(), tc.expAudiences; !reflect.DeepEqual(got, want) {
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
			if got, want := token.Subject(), tc.expSubject; got != want {
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

func (j *fakeJVS) CreateJustification(ctx context.Context, req *jvspb.CreateJustificationRequest) (*jvspb.CreateJustificationResponse, error) {
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
		Subject(req.GetSubject()).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	if err := jvspb.SetJustifications(token, req.GetJustifications()); err != nil {
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
