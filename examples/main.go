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
	mux.HandleFunc("/internal-redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/one", http.StatusFound)
	})
	mux.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) {
		search := r.FormValue("search")
		if err := r.ParseForm(); err != nil {
			panic(err)
		}

		switch r.Method {
		case http.MethodGet:
			if r.Header.Get("X-Requested-With") != "" {
				w.Header().Set("Content-Type", "text/javascript")
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.WriteHeader(http.StatusOK)

				js := []byte(`document.querySelector("#output").append("` + template.JSEscapeString(search) + `\n")`)
				w.Write(js)
				return
			}

			if err := t.ExecuteTemplate(w, "search.html", search); err != nil {
				panic(err)
			}

		case http.MethodPost:
			http.Redirect(w, r, "/form?search="+search, http.StatusFound)
		}
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
