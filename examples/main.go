package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/bentranter/go-turbolinks"
)

func main() {
	t, err := template.ParseGlob("templates/*")
	if err != nil {
		panic(err)
	}

	fmt.Println(t.DefinedTemplates())

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := t.ExecuteTemplate(w, "index.html", nil); err != nil {
			panic(err)
		}
	})
	mux.HandleFunc("/one", func(w http.ResponseWriter, r *http.Request) {
		if err := t.ExecuteTemplate(w, "one.html", nil); err != nil {
			panic(err)
		}
	})
	mux.HandleFunc("/external-redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://github.com", http.StatusFound)
	})
	mux.HandleFunc("/dist/main.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "dist/main.js")
	})
	mux.HandleFunc("/css/style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "css/style.css")
	})

	fmt.Println("Started on http://localhost:3000")
	http.ListenAndServe(":3000", turbolinks.Middleware(mux))
}
