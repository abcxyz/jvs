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

package ui

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// Pair represents a key value pair used by the select HTML element.
type Pair struct {
	Key  string
	Text string
}

// Content defines the displayable parts of the token retrieval form.
type Content struct {
	UserLabel     string
	CategoryLabel string
	ReasonLabel   string
	TTLLabel      string
	Categories    []Pair
	TTLs          []Pair
}

// FormDetails represents all the input and content used for the token retrievlal form.
type FormDetails struct {
	WindowName string
	Origin     string
	PageTitle  string
	Content    Content
	Category   string
	Reason     string
	TTL        string
	Errors     map[string]string
}

// SuccessDetails represents the data used for the success page and the postMessage response to the client.
type SuccessDetails struct {
	PageTitle    string
	Token        string
	TargetOrigin string
	WindowName   string
}

// ForbiddenDetails represents the data used for the forbidden page.
type ForbiddenDetails struct {
	PageTitle string
	Message   string
}

// Server holds the parsed html templates.
type Server struct {
	templates map[string]*template.Template
	allowList []string
}

var (
	categories = []string{"explanation", "breakglass"}
	ttls       = []string{"15", "30", "60", "120", "240"}
)

// NewServer creates a new HTTP server implementation that will handle
// rendering the JVS form and parses the go templates.
func NewServer(ctx context.Context, cfg *ServiceConfig, tmplLocations map[string]string) (*Server, error) {
	templateMap := make(map[string]*template.Template)

	// Parse templates
	for key, path := range tmplLocations {
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			return nil, fmt.Errorf("parsing %s template: %w", path, err)
		}
		templateMap[key] = tmpl
	}

	return &Server{
		templates: templateMap,
		allowList: cfg.AllowList,
	}, nil
}

// Routes creates a ServeMux of all of the routes that
// this Router supports.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./assets/static"))
	mux.Handle("/assets/static/", http.StripPrefix("/assets/static/", fs))
	mux.Handle("/popup", s.handlePopup())
	return mux
}

func (s *Server) handlePopup() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlePopupFunc(w, r, s.templates, s.allowList)
	})
}

// handlePopupFunc the initial page load of the form as well as responds to the submission of the form and ultimately renders an HTML page.
func handlePopupFunc(w http.ResponseWriter, r *http.Request, templates map[string]*template.Template, allowList []string) {
	formDetails := getFormDetails(r)

	// Initial page load, just render the page
	if r.Method == "GET" {
		// set some defaults for the form
		formDetails.Category = categories[0]
		formDetails.TTL = ttls[0]
		render(w, templates["popup"], formDetails)
		return
	}

	// Form submission
	if r.Method == "POST" {
		// 1. Check if the origin is part of the allowlist
		if !validateOrigin(r.FormValue("origin"), allowList) {
			forbiddenDetails := &ForbiddenDetails{
				PageTitle: "JVS - Error page",
				Message:   "Something went wrong",
			}
			render(w, templates["forbidden"], forbiddenDetails)
			return
		}

		// 2. Validate input
		if !validateForm(formDetails) {
			render(w, templates["popup"], formDetails)
			return
		}

		// 3. [TODO] Request a token
		token := "token_from_server"

		// 4. Redirect to a confirmation page with context, ultimately needed to postMessage back to the client
		successDetails := &SuccessDetails{
			PageTitle:    "JVS - Successful token retrieval",
			Token:        token,
			TargetOrigin: formDetails.Origin,
			WindowName:   formDetails.WindowName,
		}

		render(w, templates["success"], successDetails)
	}
}

// Checks the origin parameter against all entries in the allow list.
func validateOrigin(originParam string, allowList []string) bool {
	// origin: shop.acme.com
	// allowList: [acme.com,*]

	// Check if origin is localhost or private ip
	if strings.HasPrefix(originParam, "http://localhost") {
		return true
	}

	originSplit := strings.Split(originParam, ".")

	for _, domain := range allowList {
		// special case, allow everything
		if domain == "*" {
			return true
		}

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
				return true
			}
		}
	}
	return false
}

func validateForm(formDetails *FormDetails) bool {
	formDetails.Errors = make(map[string]string)

	if !isValidOneOf(formDetails.Category, categories) {
		formDetails.Errors["Category"] = "Category must be selected"
	}

	if strings.TrimSpace(formDetails.Reason) == "" {
		formDetails.Errors["Reason"] = "Reason is required"
	}

	if !isValidOneOf(formDetails.TTL, ttls) {
		formDetails.Errors["TTL"] = "TTL is required"
	}

	return len(formDetails.Errors) == 0
}

func isValidOneOf(selection string, options []string) bool {
	for _, v := range options {
		if v == selection {
			return true
		}
	}
	return false
}

func getFormDetails(r *http.Request) *FormDetails {
	return &FormDetails{
		WindowName: r.FormValue("windowname"),
		Origin:     r.FormValue("origin"),
		Category:   r.FormValue("category"),
		Reason:     r.FormValue("reason"),
		TTL:        r.FormValue("ttl"),
		PageTitle:  "JVS - Justification Request System",
		Content: Content{
			UserLabel:     "User",
			CategoryLabel: "Category",
			ReasonLabel:   "Reason",
			TTLLabel:      "TTL",
			Categories: []Pair{
				{
					Key:  "explanation",
					Text: "Explanation",
				},
				{
					Key:  "breakglass",
					Text: "Breakglass",
				},
			},
			TTLs: []Pair{
				{
					Key:  "15",
					Text: "15m",
				},
				{
					Key:  "30",
					Text: "30m",
				},
				{
					Key:  "60",
					Text: "1h",
				},
				{
					Key:  "120",
					Text: "2h",
				},
				{
					Key:  "240",
					Text: "4h",
				},
			},
		},
	}
}

func render(w http.ResponseWriter, tmpl *template.Template, data any) {
	if err := tmpl.Execute(w, data); err != nil {
		log.Print(err)
		http.Error(w, "Sorry, something went wrong", http.StatusInternalServerError)
	}
}
