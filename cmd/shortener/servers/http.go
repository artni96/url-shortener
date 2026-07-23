package servers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

// HTTPServer represents the HTTP server instance for the app.
type HTTPServer struct {
	app    *config.App
	r      *chi.Mux
	server *http.Server
}

// InitHTTPServer initializes a new HTTP server.
func (s *HTTPServer) InitHTTPServer() {
	s.server = &http.Server{
		Addr:    s.app.Cfg.ServerAddress,
		Handler: s.r,
	}
}

// RunHTTPServer launches the HTTP server.
func (s *HTTPServer) RunHTTPServer() error {
	s.InitHTTPServer()

	if s.app.Cfg.EnableHTTPS {
		if s.app.Cfg.Mode == "prod" {
			manager := &autocert.Manager{
				Cache:      autocert.DirCache("cache"),
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(s.app.Cfg.HostWhitelist...),
			}
			s.server.TLSConfig = manager.TLSConfig()
		}

		err := s.server.ListenAndServeTLS(s.app.Cfg.CertFile, s.app.Cfg.KeyFile)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.app.Logger.Error("failed to start HTTPS server", zap.Error(err))
			return err
		}
		return nil
	}

	err := s.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.app.Logger.Error("failed to start HTTP server", zap.Error(err))
		return err
	}
	return nil
}

// Shutdown stops the active HTTP server.
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	err := s.server.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}
	return nil
}

// NewHTTPServer returns a new HTTP server instance.
func NewHTTPServer(app *config.App, r *chi.Mux) *HTTPServer {
	return &HTTPServer{
		app: app,
		r:   r,
	}
}
