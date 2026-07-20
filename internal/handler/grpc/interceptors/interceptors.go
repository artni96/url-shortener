package interceptors

import (
	"context"
	"net"
	"strings"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func AuthInterceptor(app *config.App) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod == "/urlsProto.ShortenerService/Login" || info.FullMethod == "/urlsProto.ShortenerService/ExpandURL" {
			return handler(ctx, req)
		}

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			token := md.Get("authorization")[0]
			if token != "" {
				userID := service.GetUserID(token, app.Cfg)
				if userID == -1 {
					app.Logger.Info("user is not authorized via jwt token")
					return nil, status.Errorf(codes.Unauthenticated, "invalid token")
				}
				ctx = context.WithValue(ctx, "user_id", userID)
				return handler(ctx, req)
			}
		}
		return handler(ctx, req)
	}
}

func SubnetInterceptor(app *config.App) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		strIP, ok := peer.FromContext(ctx)
		if !ok {
			app.Logger.Info("failed to extract ip from context")
			return nil, status.Errorf(codes.PermissionDenied, "no peer found")
		}

		ip := net.ParseIP(strings.Split(strIP.Addr.String(), ":")[0])
		if ip == nil {
			app.Logger.Info("ip is not defined")
			return nil, status.Errorf(codes.PermissionDenied, "invalid ip")
		}

		_, subnet, err := net.ParseCIDR(app.Cfg.TrustedSubnet)
		if err != nil {
			app.Logger.Info("failed to parse subnet")
			return nil, status.Errorf(codes.PermissionDenied, "invalid subnet")
		}
		if subnet.Contains(ip) {
			return handler(ctx, req)
		}
		app.Logger.Info("ip does not belong to the trusted subnet")
		return nil, status.Errorf(codes.PermissionDenied, "ip does not belong to the trusted subnet")
	}
}
