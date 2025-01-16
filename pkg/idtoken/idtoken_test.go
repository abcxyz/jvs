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

package idtoken

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"golang.org/x/oauth2"

	"github.com/abcxyz/pkg/testutil"
)

type fakeDefaultTokenSource struct {
	expiry    time.Time
	idToken   string
	returnErr error
}

func (ts *fakeDefaultTokenSource) Token() (*oauth2.Token, error) {
	if ts.returnErr != nil {
		return nil, ts.returnErr
	}

	token := &oauth2.Token{
		Expiry: ts.expiry,
	}

	return token.WithExtra(url.Values{
		"id_token": {ts.idToken},
	}), nil
}

func TestIDTokenFromDefaultTokenSource(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name      string
		ts        *fakeDefaultTokenSource
		wantToken *oauth2.Token
		wantErr   string
	}{{
		name: "success",
		ts: &fakeDefaultTokenSource{
			expiry:  now,
			idToken: "id-token",
		},
		wantToken: &oauth2.Token{
			AccessToken: "id-token",
			Expiry:      now,
		},
	}, {
		name: "error",
		ts: &fakeDefaultTokenSource{
			expiry:    now,
			returnErr: fmt.Errorf("token err"),
		},
		wantErr: "token err",
	}}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ts := &tokenSource{tokenSource: tc.ts}
			gotToken, err := ts.Token()
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("unexpected err: %s", diff)
			}
			if diff := cmp.Diff(tc.wantToken, gotToken, cmpopts.IgnoreUnexported(oauth2.Token{})); diff != "" {
				t.Errorf("ID token (-want,+got):\n%s", diff)
			}
		})
	}
}

// Log in as an end user with gcloud.
// `gcloud auth application-default login`
// Set env var MANUAL_TEST=true to run the test.
func TestFromDefaultCredentials(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	if os.Getenv("MANUAL_TEST") == "" {
		t.Skip("Skip manual test; set env var MANUAL_TEST to enable")
	}

	ts, err := FromDefaultCredentials(ctx)
	if err != nil {
		t.Fatalf("failed to get ID token source: %v", err)
	}

	tk, err := ts.Token()
	if err != nil {
		t.Fatalf("failed to get ID token: %v", err)
	}

	if _, err := jwt.ParseInsecure([]byte(tk.AccessToken),
		jwt.WithContext(ctx),
		jwt.WithAcceptableSkew(5*time.Second),
	); err != nil {
		t.Errorf("%q not a valid ID token: %v", tk.AccessToken, err)
	}
}
