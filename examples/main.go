package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/bentranter/go-turbolinks"
)

func main() {
	t, err := template.ParseGlob("templates")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := t.ExecuteTemplate(w, "index", nil); err != nil {
			panic(err)
		}
	})

	fmt.Println("Started on http://localhost:3000")
	http.ListenAndServe(":3000", turbolinks.Middleware(mux))
}
