package main

import (
	"net/http"

	"github.com/artni96/url-shortener/internal/handler/urls"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", urls.CreateURL)
	mux.HandleFunc("/{id}", urls.GetURL)
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}
