package data

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository"
	"go.uber.org/zap"
)

type Reader struct {
	file    *os.File
	scanner *bufio.Scanner
}

func NewReader(filename string) (*Reader, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return &Reader{file: file, scanner: bufio.NewScanner(file)}, nil
}

func (r *Reader) Close() error {
	return r.file.Close()
}

func (r *Reader) CollectData() ([]model.URLEntity, error) {
	var result []model.URLEntity
	for r.scanner.Scan() {

		data := r.scanner.Bytes()

		object := model.URLEntity{}
		if err := json.Unmarshal(data, &object); err != nil {
			return nil, err
		}
		result = append(result, object)
	}
	return result, nil
}

func UploadFileData(filepath string, repo *repository.LocalURLRepository, log *zap.Logger) error {
	fileReader, err := NewReader(filepath)
	defer func(fileReader *Reader) {
		err := fileReader.Close()
		if err != nil {
			log.Fatal("",
				zap.Error(err),
			)
		}
	}(fileReader)

	if err != nil {
		log.Fatal(err.Error())
		return err
	}
	result, err := fileReader.CollectData()
	if err != nil {
		log.Fatal(err.Error())
		return err
	}
	for _, object := range result {
		err := repo.SaveURL(object)
		if err != nil {
			log.Fatal(err.Error())
			return err
		}
	}

	return nil
}
