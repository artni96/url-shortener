package urls

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

type ErrorResponseStruct struct {
	Error string `json:"error"`
}

func ErrorResponse(w http.ResponseWriter, errMessage string, statusCode int, logger *zap.Logger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errResp := ErrorResponseStruct{
		Error: errMessage,
	}
	logger.Error(errMessage)
	if statusCode >= 500 {
		w.Write([]byte(`{"error": "invalid request"}`))
	} else {
		resp, err := json.Marshal(errResp)
		if err != nil {
			w.Write([]byte(`{"error": "invalid request"}`))
		}
		w.Write(resp)
	}
}
