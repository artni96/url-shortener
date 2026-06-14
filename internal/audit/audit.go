package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/artni96/url-shortener/internal/model"
	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/config"
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

// Semaphore is an implementation of Semaphore pattern via a semeCh channel.
type Semaphore struct {
	semaCh chan struct{}
}

// NewSemaphore initializes a new Semaphore.
func NewSemaphore(limit int) *Semaphore {
	return &Semaphore{
		semaCh: make(chan struct{}, limit),
	}
}

// Acquire fills up semaCh with an object.
func (s *Semaphore) Acquire() {
	s.semaCh <- struct{}{}
}

// Release read an object from semaCh.
func (s *Semaphore) Release() {
	<-s.semaCh
}

// RunAudit sends Audit entities according to the cfg.
func RunAudit(app *config.App) {
	var sinks []AuditSink

	var wg sync.WaitGroup
	semaphore := NewSemaphore(10)

	var fileWriter *AuditWriter
	var fileSink *FileSink

	if app.Cfg.AuditFile != "" {
		var err error
		fileWriter, err = NewAuditWriter(app.Cfg.AuditFile)
		if err != nil {
			app.Logger.Error("failed to initialize file writer", zap.Error(err))
		}
		fileSink = NewFileSink(app, fileWriter)
		defer fileSink.Close()
		sinks = append(sinks, fileSink)
	}

	var httpSink *HTTPSink
	if app.Cfg.AuditURL != "" {
		httpSink = NewHTTPSink(app)
		defer httpSink.Close()
		sinks = append(sinks, httpSink)
	}

	for obj := range app.AuditChan {
		wg.Add(1)
		semaphore.Acquire()

		go func(obj model.AuditEntity) {

			defer wg.Done()
			defer semaphore.Release()

			for _, sink := range sinks {
				if err := sink.Sink(obj); err != nil {
					app.Logger.Error("failed to run audit entity", zap.Error(err))
				}
			}
		}(obj)
	}
	wg.Wait()
}
