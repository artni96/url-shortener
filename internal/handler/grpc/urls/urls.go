package urls

import (
	"context"
	"errors"
	"fmt"
	"time"

	pb "github.com/artni96/url-shortener/api/proto"
	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/handler/grpc/interceptors"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/urls"
	"github.com/artni96/url-shortener/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// GRPCServerHandler is an entrypoint to the gRPC server services.
type GRPCServerHandler struct {
	pb.UnimplementedShortenerServiceServer
	URLService  service.URLService
	UserService service.UserService
	App         *config.App
}

// ExpandURL returns an original URL by its short url.
func (s *GRPCServerHandler) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	var response pb.URLExpandResponse
	shortURL := req.GetId()
	originalURL, err := s.URLService.GetByShortURL(ctx, shortURL)
	if err != nil {
		if errors.Is(err, urls.ErrURLNotFound) {
			return nil, status.Errorf(codes.NotFound, "short url not found")
		}
		return nil, err
	}
	response.SetResult(originalURL.OriginalURL)
	return &response, nil
}

// ShortenURL creates a new URL entity in the storage.
func (s *GRPCServerHandler) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	var response pb.URLShortenResponse

	userID, ok := interceptors.GetUserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to get user ID")
	}

	requestEntity := model.URLCreateRequest{
		OriginalURL: req.GetUrl(),
		CreatedBy:   userID,
	}
	shortURL, err := s.URLService.Create(ctx, requestEntity, s.App.Cfg.ResponseDomain)
	if err != nil {
		return nil, status.Errorf(codes.AlreadyExists, "failed to create shorten URL: %s", err)
	}

	response.SetResult(shortURL)

	auditEntity := model.AuditEntity{
		Ts:     time.Now().Unix(),
		URL:    req.GetUrl(),
		UserID: userID,
		Action: "shorten",
	}
	if s.App.AuditChan != nil {
		s.App.AuditChan <- auditEntity
	}
	return &response, nil
}

// ListUserURLs returns URL entities created by a user.
func (s *GRPCServerHandler) ListUserURLs(ctx context.Context, req *pb.UserURLsRequest) (*pb.UserURLsResponse, error) {
	var response pb.UserURLsResponse

	strUserID := ctx.Value("user_id")
	userID := strUserID.(int)

	userURLs, err := s.URLService.GetUserList(ctx, s.App.Cfg.ResponseDomain, userID)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "failed to list user URLs: %s", err)
	}
	var pbURLs []*pb.URLData
	for _, url := range userURLs {
		pbURL := pb.URLData{}
		pbURL.SetShortUrl(url.ShortURL)
		pbURL.SetOriginalUrl(url.OriginalURL)
		pbURLs = append(pbURLs, &pbURL)
	}
	response.SetUrl(pbURLs)
	return &response, nil
}

// Login allows a user to authorize by their IP by returning a jwt token.
func (s *GRPCServerHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	response := &pb.LoginResponse{}
	peers, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract peer from ctx")
	}
	userID, err := s.UserService.GetByIP(ctx, peers.Addr.String())
	if userID != -1 && err == nil {
		jwt, err := s.UserService.Login(userID, s.App.Cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to login: %w", err)
		}
		response.SetToken(jwt)
		return response, nil
	}

	user, err := s.UserService.Create(ctx, peers.Addr.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create user token: %w", err)
	}
	userID = user.ID
	jwt, err := s.UserService.Login(user.ID, s.App.Cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	response.SetToken(jwt)
	return response, nil
}
