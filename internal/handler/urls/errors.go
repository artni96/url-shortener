package urls

import "fmt"

func shortURLGenerationError(err error) error {
	return fmt.Errorf("ошибка при генерации короткой ссылки: %w", err)
}
