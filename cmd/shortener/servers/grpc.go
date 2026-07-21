package servers

import (
	"errors"
	"fmt"
	"net"

	pb "github.com/artni96/url-shortener/api/proto"
	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/handler/grpc/urls"
	"github.com/artni96/url-shortener/internal/service"
	"google.golang.org/grpc"
)

var ErrGRPCServerFailed = errors.New("gRPC-server failed to launch")

func NewGRPCServer(server *grpc.Server, app *config.App, urlService service.URLService, userService service.UserService) error {
	listen, err := net.Listen("tcp", app.Cfg.GRPCAddress)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGRPCServerFailed, err)
	}
	pb.RegisterShortenerServiceServer(server, &urls.GRPCServerHandler{
		URLService:  urlService,
		UserService: userService,
		App:         app,
	})
	if err = server.Serve(listen); err != nil {
		return fmt.Errorf("%w: %w", ErrGRPCServerFailed, err)
	}
	return nil
}
