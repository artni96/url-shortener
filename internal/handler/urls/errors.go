package urls

import "fmt"

func shortUrlGenerationError(err error) error {
	return fmt.Errorf("ошибка при генерации короткой ссылки: %w", err)
}
