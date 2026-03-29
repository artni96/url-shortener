package data

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository"
	"go.uber.org/zap"
)

type FileScanner struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewFileScanner(filename string) (*FileScanner, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return &FileScanner{file: file, scanner: bufio.NewScanner(file)}, nil
}

func (s *FileScanner) Close() error {
	return s.file.Close()
}

func (s *FileScanner) CollectData() ([]model.URLEntity, error) {
	var result []model.URLEntity
	for s.scanner.Scan() {

		data := s.scanner.Bytes()

		object := model.URLEntity{}
		if err := json.Unmarshal(data, &object); err != nil {
			return nil, err
		}
		result = append(result, object)
	}
	return result, nil
}

func UploadFileData(filepath string, repo *repository.LocalURLRepository, log *zap.Logger) {
	fileReader, err := NewFileScanner(filepath)
	defer func(fileReader *FileScanner) {
		err := fileReader.Close()
		if err != nil {
			log.Fatal("",
				zap.Error(err),
			)
		}
	}(fileReader)

	if err != nil {
		log.Fatal(err.Error())
	}
	result, err := fileReader.CollectData()
	if err != nil {
		log.Fatal(err.Error())
	}
	for _, object := range result {
		err := repo.SaveURL(object)
		if err != nil {
			log.Fatal(err.Error())
		}
	}
}
