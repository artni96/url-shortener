package service

import "fmt"

type FailedToCreatedError struct {
	message string
}

func (e *FailedToCreatedError) Error() string {
	return fmt.Sprintf("Не удалось создать ссылку для %s", e.message)
}
