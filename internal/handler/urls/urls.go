package urls

import (
	"crypto/rand"
	"encoding/base64"
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
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Не получается получить тело запроса"))
		return
	}
	defer r.Body.Close()

	urlStr := string(originalURL)

	urlID, err := generateID(10)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	urlID = "/" + urlID
	testDB[urlID] = urlStr
	shortURL := fmt.Sprintf("http://localhost:8080%s", urlID)
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
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", redirectTo)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func generateID(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], err
}
