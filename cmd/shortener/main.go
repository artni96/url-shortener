package main

import (
	"net/http"

	"github.com/artni96/url-shortener/internal/handler/urls"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", urls.CreateURLHandler)
	mux.HandleFunc("/{id}", urls.GetURLHandler)
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}
