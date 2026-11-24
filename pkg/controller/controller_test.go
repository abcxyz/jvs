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
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/testing/protocmp"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/internal/envtest"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/pkg/testutil"
)

type mockValidator struct {
	Valid       bool
	DisplayName string
	Hint        string
}

func (v *mockValidator) Validate(context.Context, *jvspb.ValidateJustificationRequest) (*jvspb.ValidateJustificationResponse, error) {
	return &jvspb.ValidateJustificationResponse{Valid: v.Valid}, nil
}

func (v *mockValidator) GetUIData(context.Context, *jvspb.GetUIDataRequest) (*jvspb.UIData, error) {
	return &jvspb.UIData{
		DisplayName: v.DisplayName,
		Hint:        v.Hint,
	}, nil
}

type testValidateFormParam struct {
	name   string
	detail FormDetails
	want   bool
}

func TestHandlePopup(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

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
		{
			name:        "post_rejects_query_smuggled_origin",
			method:      http.MethodPost,
			path:        "/popup",
			headers:     http.Header{iapHeaderName: []string{"accounts.google.com:test@email.com"}},
			queryParam:  &url.Values{"origin": {"https://evil.net?x=.example.com"}},
			allowlist:   []string{"*.example.com"},
			wantResCode: http.StatusBadRequest,
		},
		{
			name:        "post_rejects_localhost_not_in_allowlist",
			method:      http.MethodPost,
			path:        "/popup",
			headers:     http.Header{iapHeaderName: []string{"accounts.google.com:test@email.com"}},
			queryParam:  &url.Values{"origin": {"http://127.0.0.1"}},
			allowlist:   []string{"example.com"},
			wantResCode: http.StatusBadRequest,
		},
		{
			name:        "post_rejects_http_in_prod_mode",
			method:      http.MethodPost,
			path:        "/popup",
			headers:     http.Header{iapHeaderName: []string{"accounts.google.com:test@email.com"}},
			queryParam:  &url.Values{"origin": {"http://app.example.com"}},
			allowlist:   []string{"*.example.com"},
			wantResCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			harness := envtest.NewServerConfig(t, "9091", tc.allowlist, true)
			c, err := New(ctx, harness.Renderer, harness.Processor, tc.allowlist)
			if err != nil {
				t.Fatal(err)
			}

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
		name          string
		origin        string
		allowlist     []string
		devMode       bool
		wantRes       bool
		wantCanonical string
		wantErr       string
	}{
		{
			name:      "no_allowlist",
			origin:    "https://foo.com",
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
			name:      "missing_scheme_rejected",
			origin:    "foo.com",
			allowlist: []string{"foo.com"},
			wantRes:   false,
		},
		{
			name:      "origin_domain_no_match",
			origin:    "https://bar.com",
			allowlist: []string{"foo.com"},
			wantRes:   false,
		},
		{
			name:      "origin_subdomain_no_match",
			origin:    "https://go.foo.com",
			allowlist: []string{"bar.com", "baz.com"},
			wantRes:   false,
		},
		{
			name:          "origin_match_asterisk",
			origin:        "https://foo.com",
			allowlist:     []string{"*"},
			wantRes:       true,
			wantCanonical: "https://foo.com",
		},
		{
			name:      "asterisk_origin_rejected_even_with_wildcard_allowlist",
			origin:    "*",
			allowlist: []string{"*"},
			wantRes:   false,
		},
		{
			name:          "subdomain_origin_match",
			origin:        "https://go.foo.com",
			allowlist:     []string{"bar.com", "foo.com"},
			wantRes:       true,
			wantCanonical: "https://go.foo.com",
		},
		{
			name:          "domain_origin_match",
			origin:        "https://foo.com",
			allowlist:     []string{"bar.com", "foo.com"},
			wantRes:       true,
			wantCanonical: "https://foo.com",
		},
		{
			name:          "wildcard_subdomain_match",
			origin:        "https://app.example.com",
			allowlist:     []string{"*.example.com"},
			wantRes:       true,
			wantCanonical: "https://app.example.com",
		},
		{
			name:          "wildcard_nested_subdomain_match",
			origin:        "https://a.b.example.com",
			allowlist:     []string{"*.example.com"},
			wantRes:       true,
			wantCanonical: "https://a.b.example.com",
		},
		{
			name:      "wildcard_does_not_match_apex",
			origin:    "https://example.com",
			allowlist: []string{"*.example.com"},
			wantRes:   false,
		},
		{
			name:      "suffix_without_dot_boundary_rejected",
			origin:    "https://evilexample.com",
			allowlist: []string{"example.com", "*.example.com"},
			wantRes:   false,
		},
		{
			name:          "port_suffix_allowed_on_matched_domain",
			origin:        "https://example.com:8443",
			allowlist:     []string{"example.com"},
			wantRes:       true,
			wantCanonical: "https://example.com:8443",
		},
		{
			name:          "port_suffix_allowed_on_wildcard_domain",
			origin:        "https://app.example.com:8443",
			allowlist:     []string{"*.example.com"},
			wantRes:       true,
			wantCanonical: "https://app.example.com:8443",
		},
		{
			name:          "uppercase_scheme_and_host_canonicalized",
			origin:        "HTTPS://App.Example.COM:8443",
			allowlist:     []string{"*.example.com"},
			wantRes:       true,
			wantCanonical: "https://app.example.com:8443",
		},
		{
			name:      "http_rejected_in_prod",
			origin:    "http://app.example.com",
			allowlist: []string{"*.example.com"},
			devMode:   false,
			wantRes:   false,
		},
		{
			name:          "http_allowed_in_dev_mode_when_allowlisted",
			origin:        "http://app.example.com:8080",
			allowlist:     []string{"*.example.com"},
			devMode:       true,
			wantRes:       true,
			wantCanonical: "http://app.example.com:8080",
		},
		{
			name:      "http_rejected_in_dev_mode_when_not_allowlisted",
			origin:    "http://evil.net",
			allowlist: []string{"*.example.com"},
			devMode:   true,
			wantRes:   false,
		},
		{
			name:      "non_http_scheme_rejected",
			origin:    "javascript://example.com",
			allowlist: []string{"example.com", "*"},
			devMode:   true,
			wantRes:   false,
		},
		{
			name:      "localhost_not_auto_allowed",
			origin:    "https://localhost",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:      "http_localhost_not_auto_allowed",
			origin:    "http://localhost",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:      "local_ip_not_auto_allowed",
			origin:    "https://127.0.0.1",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:      "private_ip_not_auto_allowed",
			origin:    "https://10.0.0.1",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:          "explicitly_allowlisted_localhost_in_dev_mode",
			origin:        "http://localhost:3000",
			allowlist:     []string{"localhost"},
			devMode:       true,
			wantRes:       true,
			wantCanonical: "http://localhost:3000",
		},
		{
			name:          "explicitly_allowlisted_ip_literal",
			origin:        "https://127.0.0.1:8443",
			allowlist:     []string{"127.0.0.1"},
			wantRes:       true,
			wantCanonical: "https://127.0.0.1:8443",
		},
		{
			name:          "explicitly_allowlisted_ipv6_literal",
			origin:        "https://[::1]:8443",
			allowlist:     []string{"::1"},
			wantRes:       true,
			wantCanonical: "https://[::1]:8443",
		},
		{
			name:      "query_string_suffix_smuggling_rejected",
			origin:    "https://evil.net?x=.example.com",
			allowlist: []string{"*.example.com"},
			wantRes:   false,
		},
		{
			name:      "fragment_suffix_smuggling_rejected",
			origin:    "https://evil.net#.example.com",
			allowlist: []string{"*.example.com"},
			wantRes:   false,
		},
		{
			name:      "empty_trailing_query_rejected",
			origin:    "https://example.com?",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:      "empty_trailing_fragment_rejected",
			origin:    "https://example.com#",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:      "path_component_rejected",
			origin:    "https://example.com/path",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:      "trailing_slash_rejected",
			origin:    "https://example.com/",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:      "userinfo_prefix_rejected",
			origin:    "https://allowed.example.com@evil.net",
			allowlist: []string{"*.example.com"},
			wantRes:   false,
		},
		{
			name:      "userinfo_target_rejected",
			origin:    "https://user@example.com",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:      "trailing_dot_origin_rejected",
			origin:    "https://example.com.",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:      "trailing_dot_allowlist_entry_rejected",
			origin:    "https://evil.net",
			allowlist: []string{"example.com."},
			wantRes:   false,
		},
		{
			name:      "invalid_port_rejected",
			origin:    "https://example.com:99999",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:      "leading_zero_port_rejected",
			origin:    "https://example.com:08443",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:      "empty_port_rejected",
			origin:    "https://example.com:",
			allowlist: []string{"example.com"},
			wantRes:   false,
		},
		{
			name:          "unresolvable_allowlisted_host_succeeds_without_dns",
			origin:        "https://nonexistent.invalid.example.com",
			allowlist:     []string{"*.example.com"},
			wantRes:       true,
			wantCanonical: "https://nonexistent.invalid.example.com",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotCanonical, gotRes, err := validateOrigin(tc.origin, tc.allowlist, tc.devMode)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}
			if diff := cmp.Diff(tc.wantRes, gotRes); diff != "" {
				t.Errorf("Failed validating (-want,+got):\n%s", diff)
			}
			if diff := cmp.Diff(tc.wantCanonical, gotCanonical); diff != "" {
				t.Errorf("Unexpected canonical origin (-want,+got):\n%s", diff)
			}
		})
	}
}

func TestValidateForm(t *testing.T) {
	t.Parallel()

	var cases []*testValidateFormParam

	p := justification.NewProcessor(nil, &config.JustificationConfig{
		SignerCacheTimeout: 5 * time.Minute,
	}).WithValidators(map[string]jvspb.Validator{
		"jira": &mockValidator{
			Valid:       true,
			DisplayName: "Jira issue key",
			Hint:        "Jira Issue key under JVS project",
		},
		"git": &mockValidator{
			Valid:       false,
			DisplayName: "Git issue key",
			Hint:        "Git Issue key under JVS project",
		},
	})

	controller, err := New(t.Context(), nil, p, []string{})
	if err != nil {
		t.Fatal(err)
	}

	for category := range controller.categoryDisplayData {
		for ttl := range ttls {
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
				TTL:      defaultTTL,
			},
			want: false,
		},
		{
			name: "no_input_reason",
			detail: FormDetails{
				Category: jvspb.DefaultJustificationCategory,
				Reason:   "",
				TTL:      defaultTTL,
			},
			want: false,
		},
		{
			name: "no_input_ttl",
			detail: FormDetails{
				Category: jvspb.DefaultJustificationCategory,
				Reason:   "reason",
				TTL:      "",
			},
			want: false,
		},
	}

	cases = append(cases, sadPathCases...)

	for _, tc := range cases {
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

func TestGetCatagoriesDisplayData(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		validators map[string]jvspb.Validator
		wantRes    map[string]*jvspb.UIData
		wantErr    string
	}{
		{
			name:       "empty_validators",
			validators: make(map[string]jvspb.Validator),
			wantRes:    make(map[string]*jvspb.UIData),
		},
		{
			name: "success_when_validator_with_ui_data",
			validators: map[string]jvspb.Validator{
				"jira": &mockValidator{
					Valid:       true,
					DisplayName: "Jira issue key",
					Hint:        "Jira Issue key under JVS project",
				},
				"git": &mockValidator{
					Valid:       false,
					DisplayName: "Git issue key",
					Hint:        "Git Issue key under JVS project",
				},
			},
			wantRes: map[string]*jvspb.UIData{
				"jira": {
					DisplayName: "Jira issue key",
					Hint:        "Jira Issue key under JVS project",
				},
				"git": {
					DisplayName: "Git issue key",
					Hint:        "Git Issue key under JVS project",
				},
			},
		},
		{
			name: "success_with_default_validator_hidden",
			validators: map[string]jvspb.Validator{
				"jira": &mockValidator{
					Valid:       true,
					DisplayName: "Jira issue key",
					Hint:        "Jira Issue key under JVS project",
				},
				jvspb.DefaultJustificationCategory: jvspb.DefaultJustificationValidator,
			},
			wantRes: map[string]*jvspb.UIData{
				"jira": {
					DisplayName: "Jira issue key",
					Hint:        "Jira Issue key under JVS project",
				},
			},
		},
		{
			name: "success_with_default_validator_only",
			validators: map[string]jvspb.Validator{
				jvspb.DefaultJustificationCategory: jvspb.DefaultJustificationValidator,
			},
			wantRes: map[string]*jvspb.UIData{
				jvspb.DefaultJustificationCategory: {
					DisplayName: jvspb.DefaultJustificationDisplayName,
					Hint:        jvspb.DefaultJustificationHint,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotRes, err := catagoriesDisplayData(t.Context(), tc.validators)
			if diff := testutil.DiffErrString(err, tc.wantErr); diff != "" {
				t.Errorf("Unexpected err: %s", diff)
			}

			if diff := cmp.Diff(tc.wantRes, gotRes, protocmp.Transform()); diff != "" {
				t.Errorf("Display data (-want,+got):\n%s", diff)
			}
		})
	}
}
