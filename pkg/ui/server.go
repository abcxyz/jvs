package ui

import (
	"context"
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
	Token        string
	TargetOrigin string
	WindowName   string
}

var categories []string
var ttls []string

// RunServer initializes a server on port 9091 and registers a handler for the /popup route.
func RunServer(ctx context.Context) {
	categories = []string{"explanation", "breakglass"}
	ttls = []string{"15", "30", "60", "120", "240"}

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./assets/static"))
	mux.Handle("/assets/static/", http.StripPrefix("/assets/static/", fs))
	mux.HandleFunc("/popup", popup)
	log.Fatal(http.ListenAndServe(":9091", mux))
}

func popup(w http.ResponseWriter, r *http.Request) {
	details := FormDetails{
		WindowName: r.FormValue("windowname"),
		Origin:     r.FormValue("origin"),
		PageTitle:  "JVS - Justification Request System",
		Category:   r.FormValue("category"),
		Reason:     r.FormValue("reason"),
		TTL:        r.FormValue("ttl"),
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

	// Initial page load, just render the page
	if r.Method == "GET" {
		// set some defaults
		details.Category = categories[0]
		details.TTL = ttls[0]
		render(w, "./assets/templates/index.html.tmpl", details)
		return
	}

	// Form submission
	if r.Method == "POST" {
		// 1. Validate input
		if !details.Validate() {
			render(w, "./assets/templates/index.html.tmpl", details)
			return
		}

		// 2. [TODO] Request a token
		token := "token_from_server"

		// 3. Redirect to a confirmation page with context, ultimately needed to postMessage back to the client
		successContext := SuccessDetails{
			Token:        token,
			TargetOrigin: details.Origin,
			WindowName:   details.WindowName,
		}

		render(w, "./assets/templates/success.html.tmpl", successContext)
	}
}

func (formDetails *FormDetails) Validate() bool {
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

func render(w http.ResponseWriter, filename string, data interface{}) {
	tmpl, err := template.ParseFiles(filename)
	if err != nil {
		log.Print(err)
		http.Error(w, "Sorry, something went wrong", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Print(err)
		http.Error(w, "Sorry, something went wrong", http.StatusInternalServerError)
	}
}
