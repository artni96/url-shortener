package urls

import (
	"encoding/json"
	"fmt"
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
	var data string
	err = json.Unmarshal(body, &data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	fmt.Println(data)

	testDB["/EwHXdJfB"] = data

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("http://localhost:8080/EWHXdJfB")))

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
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
	fmt.Println(redirectTo)
	if redirectTo == "" {
		w.WriteHeader(500)
	}
	http.Redirect(w, r, redirectTo, http.StatusTemporaryRedirect)
}
