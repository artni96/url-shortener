package servers

import (
	"errors"
	"fmt"
	"net"

	pb "github.com/artni96/url-shortener/api/proto"
	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/handler/grpc/interceptors"
	"github.com/artni96/url-shortener/internal/handler/grpc/urls"
	"github.com/artni96/url-shortener/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var ErrGRPCServerFailed = errors.New("gRPC-server failed to launch")

// GRPCServer represents the gRPC server instance for the app.
type GRPCServer struct {
	app         *config.App
	urlService  *service.URLService
	userService *service.UserService
	server      *grpc.Server
	creds       credentials.TransportCredentials
}

// InitGRPCServer initializes a new gRPC server.
func (s *GRPCServer) InitGRPCServer() error {
	if s.app.Cfg.EnableHTTPS {
		err := s.PrepareCredentials()
		if err != nil {
			return err
		}

		s.server = grpc.NewServer(
			grpc.Creds(s.creds),
			grpc.ChainUnaryInterceptor(
				interceptors.RequestLoggerInterceptor(s.app),
				interceptors.AuthInterceptor(s.app),
				interceptors.PanicInterceptor(s.app),
			),
		)
		return nil

	}
	s.server = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.RequestLoggerInterceptor(s.app),
			interceptors.AuthInterceptor(s.app),
			interceptors.PanicInterceptor(s.app),
		),
	)

	return nil
}

// RunGRPCServer launches the gRPC server.
func (s *GRPCServer) RunGRPCServer() error {
	err := s.InitGRPCServer()
	if err != nil {
		return fmt.Errorf("failed to initialize gRPC server: %w", err)
	}

	listen, err := net.Listen("tcp", s.app.Cfg.GRPCAddress)
	if err != nil {
		return fmt.Errorf("failed to initialize listener for gRPC server: %w", err)
	}
	pb.RegisterShortenerServiceServer(s.server, &urls.GRPCServerHandler{
		URLService:  *s.urlService,
		UserService: *s.userService,
		App:         s.app,
	})
	if err = s.server.Serve(listen); err != nil {
		return fmt.Errorf("%w: %w", ErrGRPCServerFailed, err)
	}
	return nil
}

// PrepareCredentials sets tcp credentials for the gRPC server.
func (s *GRPCServer) PrepareCredentials() error {
	creds, err := credentials.NewServerTLSFromFile(s.app.Cfg.CertFile, s.app.Cfg.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to create credentials for gRPC server: %w", err)
	}
	s.creds = creds
	return nil
}

// Shutdown stops the gRPC server.
func (s *GRPCServer) Shutdown() {
	s.server.GracefulStop()
}

// NewGRPCServer returns a new gRPC server instance.
func NewGRPCServer(app *config.App, urlService *service.URLService, userService *service.UserService) *GRPCServer {
	return &GRPCServer{
		app:         app,
		urlService:  urlService,
		userService: userService,
	}
}
