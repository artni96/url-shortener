package interceptors

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type userIDCtxKey struct{}

func GetUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userIDCtxKey{}).(int)
	return userID, ok
}

func AuthInterceptor(app *config.App) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if info.FullMethod == "/urlsProto.ShortenerService/Login" || info.FullMethod == "/urlsProto.ShortenerService/ExpandURL" {
			return handler(ctx, req)
		}

		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if len(md["authorization"]) == 0 {
				app.Logger.Info("there is no authorization header in the request")
				return nil, status.Errorf(codes.Unauthenticated, "invalid token")
			}

			token := md.Get("authorization")[0]
			if token == "" {
				app.Logger.Info("authorization header is empty")
				return nil, status.Errorf(codes.Unauthenticated, "invalid token")
			}

			userID := service.GetUserID(token, app.Cfg)
			if userID == -1 {
				app.Logger.Info("user is not authorized via jwt token")
				return nil, status.Errorf(codes.Unauthenticated, "invalid token")
			}
			ctx = context.WithValue(ctx, userIDCtxKey{}, userID)
			return handler(ctx, req)

		}
		return handler(ctx, req)
	}
}

func PanicInterceptor(app *config.App) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				app.Logger.Info("got panic",
					zap.String("error message", fmt.Sprintf("panic recovered: %v\n", recovered)),
					zap.String("call stack", string(debug.Stack())),
				)
				resp = nil
				err = status.Errorf(codes.Internal, "Internal Server Error")
			}
		}()
		return handler(ctx, req)
	}
}

func RequestLoggerInterceptor(app *config.App) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		start := time.Now()
		resp, err = handler(ctx, req)
		respStatusCode := codes.OK.String()
		if err != nil {
			respStatusCode = status.Code(err).String()
		}

		duration := time.Since(start)

		app.Logger.Info("Request done",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("response_status", respStatusCode),
		)
		return resp, err
	}
}
