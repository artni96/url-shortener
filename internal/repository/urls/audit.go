package urls

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/artni96/url-shortener/internal/model"
)

// AuditWriter is an object to write Audit data into the file.
type AuditWriter struct {
	file   *os.File
	writer *bufio.Writer
	mu     sync.Mutex
}

// NewAuditWriter returns a new AuditWriter.
func NewAuditWriter(filename string) (*AuditWriter, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("could open file: %w", err)
	}
	return &AuditWriter{file: file, writer: bufio.NewWriter(file)}, nil
}

// WriteAuditEntity saves a new Audit entity into the file.
func (w *AuditWriter) WriteAuditEntity(entity model.AuditEntity) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	data, err := json.Marshal(&entity)
	if err != nil {
		return fmt.Errorf("could not marshal entity: %w", err)
	}

	if _, err = w.writer.Write(data); err != nil {
		return fmt.Errorf("could not write to file: %w", err)
	}

	if err = w.writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("could not write to file: %w", err)
	}
	defer w.Close()
	return w.writer.Flush()
}

// Close finishes AuditWriter interaction with the file.
func (w *AuditWriter) Close() error {
	return w.file.Close()
}
