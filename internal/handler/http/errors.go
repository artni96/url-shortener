package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type ErrorResponseStruct struct {
	Error string `json:"error"`
}

type ListErrorResponseStruct struct {
	Error []string `json:"error"`
}

func ErrorResponse(w http.ResponseWriter, errMessage string, statusCode int, logger *zap.Logger) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if statusCode >= 500 {
		logger.Info(errMessage)
		w.Write([]byte(`{"error": "Server error. Please try again."}`))

	} else {
		splitMessage := strings.Split(errMessage, "\n")

		if len(splitMessage) > 1 {
			var errs []string
			for _, line := range splitMessage {
				errs = append(errs, line)
			}
			resp, err := json.Marshal(ListErrorResponseStruct{errs})
			if err != nil {
				w.Write([]byte(`{"error": "invalid request"}`))
			}
			w.Write(resp)
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
}
