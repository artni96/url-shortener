package urls

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/artni96/url-shortener/internal/config/db"
)

func CreateURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Метод %s запрещен", r.Method)))
		return
	}

	originalURL := make([]byte, r.ContentLength)
	_, err := r.Body.Read(originalURL)
	if err != nil && err.Error() != "EOF" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Не получается получить тело запроса"))
		return
	}

	urlStr := string(originalURL)
	if urlStr == "" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Пустое тело запроса"))
		return
	}

	urlID, err := generateID(10)
	if err != nil || urlID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Не удалось обработать запрос"))
		return
	}

	db.LocalDB[urlID] = urlStr
	shortURL := fmt.Sprintf("http://localhost:8080/%s", urlID)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))

	defer r.Body.Close()
}

func GetURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Метод %s запрещен", r.Method)))
		return
	}
	redirectTo := db.LocalDB[strings.TrimPrefix(r.URL.Path, "/")]
	if redirectTo == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Ссылка не найдена"))
		return

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
	resp := base64.URLEncoding.EncodeToString(bytes)[:length]

	// Дополнительная проверка на уникальность сгенерированного ID
	if _, ok := db.LocalDB[resp]; ok {
		_, err = rand.Read(bytes)
		if err != nil {
			return "", err
		}
	}
	return resp, err
}
