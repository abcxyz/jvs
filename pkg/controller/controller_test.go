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

package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/internal/envtest"
	"github.com/abcxyz/pkg/testutil"
	"github.com/google/go-cmp/cmp"
)

type mockValidator struct {
	Valid bool
}

func (v *mockValidator) Validate(context.Context, *jvspb.ValidateJustificationRequest) (*jvspb.ValidateJustificationResponse, error) {
	return &jvspb.ValidateJustificationResponse{Valid: v.Valid}, nil
}

type testValidateFormParam struct {
	name   string
	detail FormDetails
	want   bool
}

func TestHandlePopup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name        string
		method      string
		path        string
		headers     http.Header
		queryParam  *url.Values
		allowlist   []string
		wantResCode int
	}{
		{
			name:        "success_get",
			method:      http.MethodGet,
			path:        "/popup",
			headers:     http.Header{iapHeaderName: []string{"acccounts.google.com:test@email.com"}},
			queryParam:  &url.Values{"origin": {"https://localhost:3000"}},
			allowlist:   []string{"*"},
			wantResCode: http.StatusOK,
		},
		{
			name:        "success_post",
			method:      http.MethodPost,
			path:        "/popup",
			headers:     http.Header{iapHeaderName: []string{"acccounts.google.com:test@email.com"}},
			queryParam:  &url.Values{"origin": {"https://localhost:3000"}},
			allowlist:   []string{"*"},
			wantResCode: http.StatusOK,
		},
		{
			name:        "invalid_query_param_attribute",
			method:      http.MethodPost,
			path:        "/popup",
			headers:     http.Header{},
			queryParam:  &url.Values{"foo": {"bar"}},
			allowlist:   []string{"*"},
			wantResCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := ctx
			harness := envtest.NewServerConfig(t, "9091", tc.allowlist, true)
			c := New(harness.Renderer, harness.Processor, tc.allowlist)

			w, r := envtest.BuildFormRequest(ctx, t, tc.method, tc.path,
				tc.queryParam)

			for key, values := range tc.headers {
				for _, value := range values {
					r.Header.Set(key, value)
				}
			}

			handler := c.HandlePopup()
			handler.ServeHTTP(w, r)

			if got, want := w.Code, tc.wantResCode; got != want {
				t.Errorf("expected %d to be %d:\n\n%s", got, want, w.Body.String())
			}
		})
	}
}

func TestValidateOrigin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		origin    string
		allowlist []string
		wantRes   bool
		wantErr   string
	}{
		{
			name:      "no_allowlist",
			origin:    "foo",
			allowlist: []string{},
			wantRes:   false,
		},
		{
			name:      "no_origin_and_no_allowlist",
			origin:    "",
			allowlist: []string{},
			wantRes:   false,
			wantErr:   "origin was not provided",
		},
		{
			name:      "no_origin",
			origin:    "",
			allowlist: []string{"foo.com"},
			wantRes:   false,
			wantErr:   "origin was not provided",
		},
		{
			name:      "origin_domain_no_match",
			origin:    "bar.com",
			allowlist: []string{"foo.com"},
			wantRes:   false,
		},
		{
			name:      "origin_subdomain_no_match",
			origin:    "go.foo.com",
			allowlist: []string{"bar.com", "baz.com"},
			wantRes:   false,
		},
		{
			name:      "origin_match_asterisk",
			origin:    "foo.com",
			allowlist: []string{"*"},
			wantRes:   true,
		},
		{
			name:      "subdomain_origin_match",
			origin:    "go.foo.com",
			allowlist: []string{"bar.com", "foo.com"},
			wantRes:   true,
		},
		{
			name:      "domain_origin_match",
			origin:    "foo.com",
			allowlist: []string{"bar.com", "foo.com"},
			wantRes:   true,
		},
		{
			name:      "localhost_origin",
			origin:    "http://localhost",
			allowlist: []string{"example.com"},
			wantRes:   true,
		},
		{
			name:      "local_ip_origin",
			origin:    "http://127.0.0.1",
			allowlist: []string{"example.com"},
			wantRes:   true,
		},
		{
			name:      "private_ip_origin",
			origin:    "http://10.0.0.1",
			allowlist: []string{"example.com"},
			wantRes:   true,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotRes, err := validateOrigin(tc.origin, tc.allowlist)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if diff := cmp.Diff(tc.wantRes, gotRes); diff != "" {
				t.Errorf("Failed validating (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestValidateLocalIp(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		origin string
		want   bool
	}{
		{
			name:   "no_origin",
			origin: "",
			want:   false,
		},
		{
			name:   "missing_protocol",
			origin: "localhost",
			want:   false,
		},
		{
			name:   "localhost_origin",
			origin: "http://localhost",
			want:   true,
		},
		{
			name:   "local_ip_origin",
			origin: "http://127.0.0.1",
			want:   true,
		},
		{
			name:   "non_local_ip_origin",
			origin: "google.com",
			want:   false,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, _ := validateLocalIP(tc.origin)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Failed validating (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestValidateForm(t *testing.T) {
	t.Parallel()

	var cases []*testValidateFormParam

	categories := categories(map[string]jvspb.Validator{
		"jira": &mockValidator{Valid: true},
		"git":  &mockValidator{Valid: false},
	})

	controller := &Controller{
		allowCategories: categories,
		categoryPairs:   categoryPairs(categories),
	}

	cats := controller.allowCategories
	ttls := ttls()

	for i := 0; i < len(cats); i++ {
		category := cats[i]
		for j := 0; j < len(ttls); j++ {
			ttl := ttls[j]
			reason := "reason"
			happyPathCase := &testValidateFormParam{
				name: fmt.Sprintf("%s_%s_%s", category, reason, ttl),
				detail: FormDetails{
					Category: category,
					Reason:   reason,
					TTL:      ttl,
				},
				want: true,
			}
			cases = append(cases, happyPathCase)
		}
	}

	sadPathCases := []*testValidateFormParam{
		{
			name: "no_input_all",
			detail: FormDetails{
				Category: "",
				Reason:   "",
				TTL:      "",
			},
			want: false,
		},
		{
			name: "no_input_category",
			detail: FormDetails{
				Category: "",
				Reason:   "reason",
				TTL:      ttls[0],
			},
			want: false,
		},
		{
			name: "no_input_reason",
			detail: FormDetails{
				Category: cats[0],
				Reason:   "",
				TTL:      ttls[1],
			},
			want: false,
		},
		{
			name: "no_input_ttl",
			detail: FormDetails{
				Category: cats[0],
				Reason:   "reason",
				TTL:      "",
			},
			want: false,
		},
	}

	cases = append(cases, sadPathCases...)

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := controller.validateForm(&tc.detail)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Failed validating (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestIsValidOneOf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		selection string
		options   []string
		want      bool
	}{
		{
			name:      "no_selection_and_options_input",
			selection: "",
			options:   []string{},
			want:      false,
		},
		{
			name:      "no_options_input",
			selection: "foo",
			options:   []string{},
			want:      false,
		},
		{
			name:      "selection_not_in_options",
			selection: "foo",
			options:   []string{"bar"},
			want:      false,
		},
		{
			name:      "selection_in_options",
			selection: "foo",
			options:   []string{"bar", "foo"},
			want:      true,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := isValidOneOf(tc.selection, tc.options)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("Failed validating (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestGetEmail(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		email   string
		wantRes string
		wantErr string
	}{
		{
			name:    "empty_email",
			email:   "",
			wantErr: "email header is not present",
		},
		{
			name:    "incorrect_format_email",
			email:   "iap-prefix/test@email.com",
			wantErr: "email value has unexpected format",
		},
		{
			name:    "happy_path1",
			email:   iapHeaderName + ":test@email.com",
			wantRes: "test@email.com",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodGet, "/popup", nil)
			r.Header.Set(iapHeaderName, tc.email)
			gotRes, err := getEmail(r)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if got, want := gotRes, tc.wantRes; got != want {
				t.Errorf("email got=%s want=%s", got, want)
			}
		})
	}
}
