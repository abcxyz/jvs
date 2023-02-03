package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type FormDetails struct {
	Category string
	Value    string
	TTL      string
}

var tmpl *template.Template

func printStuff(r *http.Request) {
	r.ParseForm()                 // parse arguments, you have to call this by yourself
	fmt.Println("r.Form", r.Form) // print form information in server side
	// set data?
	fmt.Println("r.URL.Path", r.URL.Path)
	fmt.Println("r.URL.Host", r.URL.Host)
	fmt.Println("r.URL.Hostname", r.URL.Hostname())
	fmt.Println("r.Form[\"url_long\"]", r.Form["url_long"])
	for k, v := range r.Form {
		fmt.Println("key:", k)
		fmt.Println("val:", strings.Join(v, ""))
	}
	// fmt.Fprintf(w, "Hello astaxie!") // send data to client side

	res, _ := json.Marshal(r)
	fmt.Println("r", string(res))

	fmt.Println("r.RequestURI", r.RequestURI)

	parsed, err := url.ParseRequestURI(r.RequestURI)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("parsed", parsed)

	originUriEncoded := r.URL.Query().Get("origin")
	fmt.Println("r.URL.Query().Get(\"origin\")", originUriEncoded)

	originUriDecoded, err := url.PathUnescape(originUriEncoded)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("decoded param", originUriDecoded)
}

func popup(w http.ResponseWriter, r *http.Request) {

	details := FormDetails{
		Category: r.FormValue("category"),
		Value:    r.FormValue("value"),
		TTL:      r.FormValue("ttl"),
	}

	dt := time.Now()
	fmt.Println("Current date and time is: ", dt.String())

	printStuff(r)

	tmpl.Execute(w, details)
}

func RunServer(ctx context.Context) {
	mux := http.NewServeMux()
	tmpl = template.Must(template.ParseFiles("./assets/templates/index.html"))

	fs := http.FileServer(http.Dir("./assets/static"))
	mux.Handle("/assets/static/", http.StripPrefix("/assets/static/", fs))
	mux.HandleFunc("/popup", popup)

	log.Fatal(http.ListenAndServe(":9091", mux))
}
