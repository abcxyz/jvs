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

package justification

import (
	"context"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"google.golang.org/grpc/metadata"

	"github.com/abcxyz/pkg/testutil"
)

func TestExtractRequestorFromIncomingContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name string
		ctx  context.Context //nolint:containedctx // testing
		exp  string
		err  string
	}{
		{
			name: "nil_context",
			ctx:  nil,
			exp:  "",
		},
		{
			name: "empty_context",
			ctx:  ctx,
			exp:  "",
		},
		{
			name: "no_authorization_header",
			ctx: metadata.NewIncomingContext(ctx, metadata.New(map[string]string{
				"foo": "bar",
			})),
			exp: "",
		},
		{
			name: "authorization_header_too_short",
			ctx: metadata.NewIncomingContext(ctx, metadata.New(map[string]string{
				"authorization": "abc",
			})),
			err: "invalid jwt in grpc metadata (too short)",
		},
		{
			name: "invalid_jwt",
			ctx: metadata.NewIncomingContext(ctx, metadata.New(map[string]string{
				"authorization": "bearer this-is-totally-not-a-valid-jwt",
			})),
			err: "failed to parse incoming jwt",
		},
		{
			name: "jwt_missing_email",
			ctx: metadata.NewIncomingContext(ctx, metadata.New(map[string]string{
				"authorization": "bearer " + testToken(t, nil),
			})),
			err: `missing "email" key in incoming jwt`,
		},
		{
			name: "jwt_email_not_string",
			ctx: metadata.NewIncomingContext(ctx, metadata.New(map[string]string{
				"authorization": "bearer " + testToken(t, []string{"foo@bar.com"}),
			})),
			err: `"email" key is not of type string`,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractRequestorFromIncomingContext(tc.ctx)
			if diff := testutil.DiffErrString(err, tc.err); diff != "" {
				t.Error(diff)
			}

			if got, want := got, tc.exp; got != want {
				t.Errorf("expected %q to be %q", got, want)
			}
		})
	}
}

func testToken(tb testing.TB, email any) string {
	tb.Helper()

	now := time.Now().UTC()

	t, err := jwt.NewBuilder().
		Audience([]string{"test_aud"}).
		Expiration(now.Add(5 * time.Minute)).
		JwtID(`jwt-id`).
		IssuedAt(now).
		Issuer(`test_iss`).
		NotBefore(now).
		Subject("test_sub").
		Build()
	if err != nil {
		tb.Fatalf("failed to build token: %s\n", err)
	}

	if email != nil {
		if err := t.Set("email", email); err != nil {
			tb.Fatal(err)
		}
	}

	b, err := jwt.Sign(t, jwt.WithKey(jwa.HS256, []byte("testing")))
	if err != nil {
		tb.Fatal(err)
	}
	return string(b)
}
