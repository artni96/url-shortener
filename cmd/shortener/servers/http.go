package servers

import (
	"errors"
	"net/http"

	"github.com/artni96/url-shortener/internal/config"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

// NewHTTPServer launches the server according to the app config.
func NewHTTPServer(app *config.App, server *http.Server) error {
	if app.Cfg.EnableHTTPS {
		if app.Cfg.Mode == "prod" {
			manager := &autocert.Manager{
				Cache:      autocert.DirCache("cache"),
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(app.Cfg.HostWhitelist...),
			}
			server.TLSConfig = manager.TLSConfig()
		}

		err := server.ListenAndServeTLS(app.Cfg.CertFile, app.Cfg.KeyFile)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Logger.Error("failed to start HTTPS server", zap.Error(err))
			return err
		}
		return nil
	}
	
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		app.Logger.Error("failed to start HTTP server", zap.Error(err))
		return err
	}
	return nil
}
