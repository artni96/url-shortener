package utility

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateShortURL generates a string of random bytes.
func GenerateShortURL(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("short url already exists")
	}
	resp := base64.URLEncoding.EncodeToString(bytes)[:length]
	return resp, err
}

// BulkGenerateShortURL generates several strings of random bytes.
func BulkGenerateShortURL(amount int, length int) ([]string, error) {
	result := make([]string, 0, amount)
	for range amount {
		shortURL, err := GenerateShortURL(length)
		if err != nil {
			return nil, err
		}
		result = append(result, shortURL)
	}
	return result, nil
}
