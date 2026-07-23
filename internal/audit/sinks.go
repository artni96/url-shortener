package audit

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"go.uber.org/zap"
)

type AuditSink interface {
	Sink(obj model.AuditEntity) error
	Close()
}

// generate:reset
type HTTPSink struct {
	client *http.Client
	logger *zap.Logger
	cfg    *config.Config
}

func NewHTTPSink(logger *zap.Logger, cfg *config.Config) *HTTPSink {
	return &HTTPSink{client: &http.Client{
		Timeout: 10 * time.Second,
	},
		logger: logger,
		cfg:    cfg,
	}
}

func (s *HTTPSink) Sink(obj model.AuditEntity) error {
	byteBody, err := json.Marshal(obj)
	if err != nil {
		s.logger.Error("failed to marshal audit entity", zap.Error(err))
		return err
	}
	stringBody := strings.NewReader(string(byteBody))
	req, err := http.NewRequest("POST", s.cfg.AuditURL, stringBody)
	if err != nil {
		s.logger.Error("failed to create audit entity", zap.Error(err))
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Error("failed to send audit entity", zap.Error(err))
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *HTTPSink) Close() {
	s.client.CloseIdleConnections()
}

type FileSink struct {
	logger     *zap.Logger
	fileWriter *AuditWriter
	mu         sync.Mutex
}

func NewFileSink(logger *zap.Logger, fileWriter *AuditWriter) *FileSink {
	return &FileSink{
		logger:     logger,
		fileWriter: fileWriter,
	}
}

func (s *FileSink) Sink(obj model.AuditEntity) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.fileWriter.WriteAuditEntity(obj)
	if err != nil {
		s.logger.Error("failed to write audit entity", zap.Error(err))
		return err
	}
	return nil
}

func (s *FileSink) Close() {
	s.fileWriter.Close()
}
