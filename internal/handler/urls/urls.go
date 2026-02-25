package urls

import (
	"net/http"
)

type postResponse struct {
}

func CreateURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Method not allowed"))
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Created"))
}

func GetURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Method not allowed"))
		return
	}

	w.Header().Set("Location", "https://practicum.yandex.ru/ ")
	w.WriteHeader(http.StatusTemporaryRedirect)
}
