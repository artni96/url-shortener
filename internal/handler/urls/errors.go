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

	if statusCode >= 500 {
		logger.Error(errMessage)
		w.Write([]byte(`{"error": "Server error. Please try again."}`))

	} else {
		errResp := ErrorResponseStruct{
			Error: errMessage,
		}
		resp, err := json.Marshal(errResp)
		if err != nil {
			w.Write([]byte(`{"error": "invalid request"}`))
		}
		w.Write(resp)
	}
}
