package service

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/artni96/url-shortener/internal/model"
	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/config"
	urlrepo "github.com/artni96/url-shortener/internal/repository/urls"
)

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
	var wg sync.WaitGroup
	semaphore := NewSemaphore(10)

	var fileWriter *urlrepo.AuditWriter
	if app.Cfg.AuditFile != "" {
		var err error
		fileWriter, err = urlrepo.NewAuditWriter(app.Cfg.AuditFile)
		if err != nil {
			app.Logger.Error("failed to initialize file writer", zap.Error(err))
		}
		defer func() {
			if fileWriter != nil {
				fileWriter.Close()
			}
		}()
	}

	var client *http.Client
	if app.Cfg.AuditURL != "" {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	for obj := range app.AuditChan {
		wg.Add(1)
		semaphore.Acquire()

		go func(obj model.AuditEntity) {

			if app.Cfg.AuditFile != "" {
				err := fileWriter.WriteAuditEntity(obj)
				if err != nil {
					app.Logger.Error("failed to write audit entity", zap.Error(err))
				}
			}

			if app.Cfg.AuditURL != "" && client != nil {
				byteBody, err := json.Marshal(obj)
				if err != nil {
					app.Logger.Error("failed to marshal audit entity", zap.Error(err))
				}
				stringBody := strings.NewReader(string(byteBody))
				req, err := http.NewRequest("POST", app.Cfg.AuditURL, stringBody)
				req.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(req)
				if err != nil {
					app.Logger.Error("failed to send audit entity", zap.Error(err))
				}
				defer resp.Body.Close()

			}

			defer wg.Done()
			defer semaphore.Release()
		}(obj)

	}
	wg.Wait()
}
