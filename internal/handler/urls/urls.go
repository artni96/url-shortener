package urls

import (
	"net/http"
)

type postResponse struct {
}

func CreateURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
}

func GetURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
	}
	w.WriteHeader(http.StatusTemporaryRedirect)
	w.Header().Set("Location", "https://practicum.yandex.ru/")
}
