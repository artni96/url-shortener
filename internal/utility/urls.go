package utility

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func GenerateID(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("Данный urlID уже используется")
	}
	resp := base64.URLEncoding.EncodeToString(bytes)[:length]
	return resp, err
}
