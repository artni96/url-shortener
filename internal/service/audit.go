package service

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	urlrepo "github.com/artni96/url-shortener/internal/repository/urls"
	"go.uber.org/zap"
)

type Semaphore struct {
	semaCh chan struct{}
}

func NewSemaphore(limit int) *Semaphore {
	return &Semaphore{
		semaCh: make(chan struct{}, limit),
	}
}

func (s *Semaphore) Acquire() {
	s.semaCh <- struct{}{}
}

func (s *Semaphore) Release() {
	<-s.semaCh
}

func RunAudit(app *config.App) {
	var wg sync.WaitGroup

	semaphore := NewSemaphore(10)
	for obj := range app.AuditChan {
		wg.Add(1)
		semaphore.Acquire()
		go func() {
			auditFile := app.Cfg.AuditFile
			if auditFile != "" {
				go func() {
					fileWriter, err := urlrepo.NewAuditWriter(auditFile)
					if err != nil {
						app.Logger.Error("failed to initialize file writer", zap.Error(err))
					}
					err = fileWriter.WriteAuditEntity(obj)
					if err != nil {
						app.Logger.Error("failed to write audit entity", zap.Error(err))
					}
					defer fileWriter.Close()
				}()
			}
			auditURL := app.Cfg.AuditURL
			if auditURL != "" {
				go func() {
					client := &http.Client{
						Timeout: 10 * time.Second,
					}
					byteBody, err := json.Marshal(obj)
					if err != nil {
						app.Logger.Error("failed to marshal audit entity", zap.Error(err))
					}
					stringBody := strings.NewReader(string(byteBody))
					req, err := http.NewRequest("POST", auditURL, stringBody)
					req.Header.Set("Content-Type", "application/json")

					resp, err := client.Do(req)
					if err != nil {
						app.Logger.Error("failed to send audit entity", zap.Error(err))
					}
					defer resp.Body.Close()
				}()
			}
			defer wg.Done()
			defer semaphore.Release()
		}()
		wg.Wait()
	}
}
