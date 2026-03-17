package urls

import (
	"encoding/json"
	"net/http"
)

type ErrorResponseStruct struct {
	Error string `json:"error"`
}

func ErrorResponse(w http.ResponseWriter, errMessage string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	errResp := ErrorResponseStruct{
		Error: errMessage,
	}
	resp, err := json.Marshal(errResp)
	if err != nil {
		w.Write([]byte(`{"error": "invalid request"}`))
	}
	w.Write(resp)
}
