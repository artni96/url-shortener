package utility

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/artni96/url-shortener/internal/repository"
	"go.uber.org/zap"
)

func GenerateShortURL(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("short url already exists")
	}
	resp := base64.URLEncoding.EncodeToString(bytes)[:length]
	return resp, err
}

func BulkGenerateShortURL(amount int, length int) ([]string, error) {
	var result []string
	for range amount {
		shortURL, err := GenerateShortURL(length)
		if err != nil {
			return nil, err
		}
		result = append(result, shortURL)
	}
	return result, nil
}

func RemoveEntityFromFile(filepath string, shortURL string, logger *zap.Logger) error {
	fileReader, err := repository.NewFileScanner(filepath)
	if fileReader == nil {
		return nil
	}
	defer func(fileReader *repository.FileScanner) {
		err := fileReader.Close()
		if err != nil {
			logger.Info("could not close file reader", zap.String("filepath", filepath))
		}
	}(fileReader)

	if err != nil {
		logger.Info("could not initialize NewFileScanner",
			zap.String("error", err.Error()))
		return nil
	}
	if filepath != "" {
		filteredEntities, err := fileReader.CollectFilteredData(shortURL)
		if err != nil {
			logger.Info("could not collect data from file",
				zap.String("filepath", filepath),
				zap.String("error", err.Error()))
			return fmt.Errorf("could not collect data from file: %w", err)
		}

		fileWriter, err := repository.NewWriter(filepath)
		defer fileWriter.Close()

		if err != nil {
			logger.Error("could not create file writer",
				zap.String("path", filepath),
				zap.String("error message", err.Error()),
			)
			return fmt.Errorf("could not create file writer: %w", err)
		}

		err = fileWriter.BulkWriteEntities(filteredEntities, true)
		if err != nil {
			logger.Info("could not rewrite file after url removal",
				zap.String("path", filepath),
				zap.String("removed entity", shortURL))
			return fmt.Errorf("%w", err)
		}
	}
	return nil
}
