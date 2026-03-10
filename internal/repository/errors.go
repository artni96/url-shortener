package repository

import "fmt"

type DuplicateURLIDError struct {
	message string
}

func (e DuplicateURLIDError) Error() string {
	return fmt.Sprintf("%s найден в БД", e.message)
}
