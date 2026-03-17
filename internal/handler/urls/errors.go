package urls

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/artni96/url-shortener/internal/model"
)

func shortURLGenerationError(err error) error {
	return fmt.Errorf("ошибка при генерации короткой ссылки: %w", err)
}

func ErrorResponse(w http.ResponseWriter, errMessage string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	errResp := model.ErrorResponse{
		Error: errMessage,
	}
	resp, err := json.Marshal(errResp)
	if err != nil {
		w.Write([]byte(`{"error": "invalid request"}`))
	}
	w.Write(resp)
}
