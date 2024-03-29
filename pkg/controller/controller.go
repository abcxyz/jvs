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
	"strings"
	"time"

	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/durationpb"

	jvspb "github.com/abcxyz/jvs/apis/v0"
	"github.com/abcxyz/jvs/internal/project"
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
	}, nil
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
	origin := r.FormValue("origin")
	if validOrigin, err := validateOrigin(origin, c.allowlist); err != nil || !validOrigin {
		var m string
		if err != nil {
			m = err.Error()
		} else if !validOrigin {
			m = "Unexpected origin provided"
		}

		c.renderBadRequest(w, m)
		return
	}

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

// Checks the origin parameter against all entries in the allow list.
func validateOrigin(originParam string, allowlist []string) (bool, error) {
	if len(originParam) == 0 {
		return false, fmt.Errorf("origin was not provided")
	}

	// Check if origin is localhost or private ip
	validIP, err := validateLocalIP(originParam)
	if err != nil {
		return false, err
	}

	// either local development or all origins are allowed
	if validIP || (len(allowlist) == 1 && allowlist[0] == "*") {
		return true, nil
	}

	originSplit := strings.Split(originParam, ".")

	for _, domain := range allowlist {
		domainSplit := strings.Split(domain, ".")

		// this domain is longer than the origin, skip over it
		if len(domainSplit) > len(originSplit) {
			continue
		}

		// compare the origin and current domain from right to left
		for i, j := len(domainSplit)-1, len(originSplit)-1; i >= 0 && j >= 0; i, j = i-1, j-1 {
			// not a wildcard reference and no match, proceed to the next domain candidate in the allow list
			if domainSplit[i] != "*" && domainSplit[i] != originSplit[j] {
				break
			}

			if i == 0 {
				return true, nil
			}
		}
	}

	return false, nil
}

func validateLocalIP(originParam string) (bool, error) {
	if project.DevMode() {
		return true, nil
	}

	u, err := url.Parse(originParam)
	if err != nil {
		return false, fmt.Errorf("unable to parse url: %w", err)
	}

	ipAddr, err := net.ResolveIPAddr("ip", u.Hostname())
	if err != nil {
		return false, fmt.Errorf("unable to resolve IP Address: %w", err)
	}

	parsedIP := net.ParseIP(ipAddr.IP.String())
	return (parsedIP.IsLoopback() || parsedIP.IsPrivate()), nil
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
