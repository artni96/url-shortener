package urls

import (
	"io"
	"net/http"
)

var testDB = map[string]string{}

func CreateURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Method not allowed"))
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	testDB["/EwHXdJfB"] = string(body)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("http://localhost:8080/EWHXdJfB"))

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(r.Body)
}

func GetURL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Method not allowed"))
		return
	}
	redirectTo := testDB[r.URL.Path]

	w.Header().Set("Location", redirectTo)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
