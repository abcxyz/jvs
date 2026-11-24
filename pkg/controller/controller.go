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
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/durationpb"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/internal/project"
	"github.com/abcxyz/jvs/pkg/config"
	"github.com/abcxyz/jvs/pkg/justification"
	"github.com/abcxyz/pkg/renderer"
)

var ttls = map[string]struct{}{
	"15m": {},
	"30m": {},
	"1h":  {},
	"2h":  {},
	"4h":  {},
}

const defaultTTL = "15m"

// Controller manages use of the renderer in the http handler.
type Controller struct {
	h                   *renderer.Renderer
	p                   *justification.Processor
	allowlist           []string
	categoryDisplayData map[string]*jvspb.UIData
	devMode             bool
}

// Content defines the displayable parts of the token retrieval form.
type Content struct {
	UserLabel     string
	CategoryLabel string
	ReasonLabel   string
	TTLLabel      string
	Categories    map[string]*jvspb.UIData
	TTLs          map[string]struct{}
}

// FormDetails represents all the input and content used for the token retrievlal form.
type FormDetails struct {
	WindowName  string
	Origin      string
	PageTitle   string
	Description string
	UserEmail   string
	Content     *Content
	Category    string
	Reason      string
	TTL         string
	Errors      map[string]string
}

// SuccessDetails represents the data used for the success page and the postMessage response to the client.
type SuccessDetails struct {
	WindowName  string
	Origin      string
	PageTitle   string
	Description string
	Token       string
}

// ErrorDetails represents the data used for the 400 page.
type ErrorDetails struct {
	PageTitle   string
	Description string
	Message     string
}

const iapHeaderName = "x-goog-authenticated-user-email"

func New(ctx context.Context, h *renderer.Renderer, p *justification.Processor, allowlist []string) (*Controller, error) {
	categories, err := catagoriesDisplayData(ctx, p.Validators())
	if err != nil {
		return nil, err
	}

	return &Controller{
		h:                   h,
		p:                   p,
		allowlist:           allowlist,
		categoryDisplayData: categories,
		devMode:             project.DevMode(),
	}, nil
}

// WithDevMode configures whether development mode (allowing http:// origins) is enabled.
func (c *Controller) WithDevMode(devMode bool) *Controller {
	c.devMode = devMode
	return c
}

func (c *Controller) HandleHealth() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.h.RenderJSON(w, http.StatusOK, nil)
	})
}

func (c *Controller) HandlePopup() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			c.handlePopupGet(w, r)
		case http.MethodPost:
			c.handlePopupPost(w, r)
		default:
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
		}
	})
}

// handlePopupGet handles the initial page load.
func (c *Controller) handlePopupGet(w http.ResponseWriter, r *http.Request) {
	formDetails, err := c.getFormDetails(r)
	if err != nil {
		c.renderBadRequest(w, err.Error())
		return
	}

	// set some defaults for the form
	if formDetails.Category == "" {
		formDetails.Category = jvspb.DefaultJustificationCategory
	}
	if formDetails.TTL == "" {
		formDetails.TTL = defaultTTL
	}

	c.h.RenderHTML(w, "popup.html", formDetails)
}

// handlePopupPost handles form submission.
func (c *Controller) handlePopupPost(w http.ResponseWriter, r *http.Request) {
	formDetails, err := c.getFormDetails(r)
	if err != nil {
		c.renderBadRequest(w, err.Error())
		return
	}

	// 1. Check if the origin is part of the allowlist
	canonicalOrigin, validOrigin, err := validateOrigin(formDetails.Origin, c.allowlist, c.devMode)
	if err != nil || !validOrigin {
		var m string
		if err != nil {
			m = err.Error()
		} else if !validOrigin {
			m = "Unexpected origin provided"
		}

		c.renderBadRequest(w, m)
		return
	}
	formDetails.Origin = canonicalOrigin

	// 2. Validate input
	if !c.validateForm(formDetails) {
		c.h.RenderHTML(w, "popup.html", formDetails)
		return
	}

	// 3. Request a token
	dur, err := time.ParseDuration(formDetails.TTL)
	if err != nil {
		c.renderBadRequest(w, err.Error())
		return
	}

	req := &jvspb.CreateJustificationRequest{
		Justifications: []*jvspb.Justification{
			{
				Category: formDetails.Category,
				Value:    formDetails.Reason,
			},
		},
		Ttl: durationpb.New(dur),
	}

	token, err := c.p.CreateToken(context.Background(), formDetails.UserEmail, req)
	if err != nil {
		c.renderBadRequest(w, err.Error())
		return
	}

	// 4. Redirect to a confirmation page with context, ultimately needed to postMessage back to the client
	successDetails := &SuccessDetails{
		PageTitle:   "JVS - Successful token retrieval",
		Description: "Successful token page",
		Token:       string(token),
		Origin:      formDetails.Origin,
		WindowName:  formDetails.WindowName,
	}
	c.h.RenderHTML(w, "success.html", successDetails)
}

// validateOrigin parses originParam as a strict web origin (scheme://host[:port]),
// enforces https (or http/https in devMode), rejects userinfo/path/query/fragment components,
// and matches the parsed hostname against allowlist without performing DNS resolution.
func validateOrigin(originParam string, allowlist []string, devMode bool) (string, bool, error) {
	if len(originParam) == 0 {
		return "", false, fmt.Errorf("origin was not provided")
	}

	// Reject raw query/fragment delimiters (including empty trailing "?" or "#"),
	// backslashes, and whitespace/control characters before url.Parse.
	if strings.ContainsAny(originParam, "?#\\\r\n\t ") {
		return "", false, nil
	}

	u, err := url.Parse(originParam)
	if err != nil {
		return "", false, fmt.Errorf("unable to parse url: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "https" && (!devMode || scheme != "http") {
		return "", false, nil
	}

	if u.Opaque != "" || u.User != nil || u.Host == "" ||
		u.Path != "" || u.RawPath != "" ||
		u.RawQuery != "" || u.ForceQuery ||
		u.Fragment != "" || u.RawFragment != "" ||
		strings.HasSuffix(u.Host, ":") {
		return "", false, nil
	}

	port := u.Port()
	if port != "" {
		if !isValidPort(port) {
			return "", false, nil
		}
	}

	host := strings.ToLower(u.Hostname())
	hostIP := net.ParseIP(host)
	if hostIP == nil && !config.IsValidHostname(host) {
		return "", false, nil
	}

	canonicalHost := host
	if port != "" {
		canonicalHost = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		canonicalHost = "[" + host + "]"
	}
	canonicalOrigin := scheme + "://" + canonicalHost

	if len(allowlist) == 1 && strings.TrimSpace(allowlist[0]) == "*" {
		return canonicalOrigin, true, nil
	}

	for _, entry := range allowlist {
		if matchAllowlistEntry(host, hostIP, entry) {
			return canonicalOrigin, true, nil
		}
	}

	return "", false, nil
}

func isValidPort(port string) bool {
	if port == "" || len(port) > 5 || port[0] == '0' {
		return false
	}
	for i := 0; i < len(port); i++ {
		if port[i] < '0' || port[i] > '9' {
			return false
		}
	}
	n, err := strconv.Atoi(port)
	return err == nil && n >= 1 && n <= 65535
}

func matchAllowlistEntry(host string, hostIP net.IP, rawEntry string) bool {
	entry := strings.ToLower(strings.TrimSpace(rawEntry))
	if entry == "" || entry == "*" {
		return false
	}

	if entryIP := net.ParseIP(entry); entryIP != nil {
		return hostIP != nil && hostIP.Equal(entryIP)
	}

	// Non-IP allowlist entries must be valid domains or wildcard domains (*.example.com).
	if err := config.ValidateAllowlistEntry(entry); err != nil {
		return false
	}
	if hostIP != nil {
		return false
	}

	if strings.HasPrefix(entry, "*.") {
		base := strings.TrimPrefix(entry, "*.")
		return strings.HasSuffix(host, "."+base) && len(host) > len(base)+1
	}

	if host == entry {
		return true
	}
	// Multi-label domain entries also permit subdomains on a dot boundary.
	if strings.Contains(entry, ".") && strings.HasSuffix(host, "."+entry) {
		return true
	}

	return false
}

func (c *Controller) validateForm(formDetails *FormDetails) bool {
	// This only does sanity check of the form, e.g. field not empty.
	// The actual verification only happens when submitting the form.
	formDetails.Errors = make(map[string]string)

	if _, ok := c.categoryDisplayData[formDetails.Category]; !ok {
		formDetails.Errors["Category"] = "Category must be selected"
	}

	if strings.TrimSpace(formDetails.Reason) == "" {
		formDetails.Errors["Reason"] = "Reason is required"
	}

	if _, ok := ttls[formDetails.TTL]; !ok {
		formDetails.Errors["TTL"] = "TTL is required"
	}

	return len(formDetails.Errors) == 0
}

func isValidOneOf(selection string, options []string) bool {
	return slices.Contains(options, selection)
}

func (c *Controller) getFormDetails(r *http.Request) (*FormDetails, error) {
	email, err := getEmail(r)
	if err != nil {
		return nil, err
	}

	return &FormDetails{
		WindowName:  r.FormValue("windowname"),
		Origin:      r.FormValue("origin"),
		Category:    c.getCategory(),
		Reason:      r.FormValue("reason"),
		UserEmail:   email,
		TTL:         r.FormValue("ttl"),
		PageTitle:   "JVS - Justification Request System",
		Description: "Justification Verification System form used for minting tokens.",
		Content: &Content{
			UserLabel:     "User",
			CategoryLabel: "Category",
			ReasonLabel:   "Reason",
			TTLLabel:      "TTL",
			Categories:    c.categoryDisplayData,
			TTLs:          ttls,
		},
	}, nil
}

// Renders a bad request page with a custom message.
func (c *Controller) renderBadRequest(w http.ResponseWriter, m string) {
	t := http.StatusText(http.StatusBadRequest)
	c.h.RenderHTMLStatus(w, http.StatusBadRequest, "400.html", &ErrorDetails{
		PageTitle:   t,
		Description: t,
		Message:     m,
	})
}

func getEmail(r *http.Request) (string, error) {
	iapEmailValue := r.Header.Get(iapHeaderName)

	if iapEmailValue == "" {
		return "", fmt.Errorf("email header is not present")
	}

	split := strings.Split(iapEmailValue, ":")
	if len(split) != 2 {
		return "", fmt.Errorf("email value has unexpected format, expected %s:domain:<email>", iapHeaderName)
	}

	return split[1], nil
}

// categoriesDisplayData gathers the plugins' display data.
func catagoriesDisplayData(ctx context.Context, validators map[string]jvspb.Validator) (map[string]*jvspb.UIData, error) {
	displayData := make(map[string]*jvspb.UIData, len(validators))

	for k, v := range validators {
		d, err := v.GetUIData(ctx, &jvspb.GetUIDataRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to get display data for category %q: %w", k, err)
		}
		displayData[k] = d
	}

	// In case there are additional category options available, we aim to hide the default option from users.
	if len(displayData) > 1 {
		delete(displayData, jvspb.DefaultJustificationCategory)
	}

	return displayData, nil
}

// The category with the lowest alphabetical value will be selected.
func (c *Controller) getCategory() string {
	// Due to the presence of the DefaultJustificationCategory, the categoryDisplayData list cannot be empty.
	keys := make([]string, 0, len(c.categoryDisplayData))
	for key := range c.categoryDisplayData {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}
