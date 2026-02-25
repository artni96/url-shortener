package urls

import (
	"fmt"
	"net/http"
)

var testDB = map[string]string{}

func CreateURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Method not allowed"))
		return
	}

	originalURL := make([]byte, r.ContentLength)
	_, err := r.Body.Read(originalURL)
	if err != nil && err.Error() != "EOF" {
		http.Error(w, "Не получается получить тело запроса", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	urlStr := string(originalURL)

	shortID := "/EwHXdJfB"

	testDB[shortID] = urlStr
	shortURL := fmt.Sprintf("http://localhost:8080/s%s", shortID)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func GetURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Method not allowed"))
		return
	}
	redirectTo := testDB[r.URL.Path]
	fmt.Println(redirectTo)
	if redirectTo == "" {
		w.WriteHeader(500)
	}
	http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
}
