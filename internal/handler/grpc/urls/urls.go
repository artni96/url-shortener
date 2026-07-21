package urls

import (
	"context"
	"fmt"

	pb "github.com/artni96/url-shortener/api/proto"
	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GRPCServerHandler is an entrypoint to the gRPC server services.
type GRPCServerHandler struct {
	pb.UnimplementedShortenerServiceServer
	URLService  service.URLService
	UserService service.UserService
	Cfg         *config.Config
}

func (s *GRPCServerHandler) ExpandURL(ctx context.Context, req *pb.URLExpandRequest) (*pb.URLExpandResponse, error) {
	var response pb.URLExpandResponse
	shortURL := req.GetId()
	originalURL, err := s.URLService.GetByShortURL(ctx, shortURL)
	if err != nil {
		return nil, err
	}
	response.SetResult(originalURL.OriginalURL)
	return &response, nil
}

func (s *GRPCServerHandler) ShortenURL(ctx context.Context, req *pb.URLShortenRequest) (*pb.URLShortenResponse, error) {
	var response pb.URLShortenResponse

	strUserID := ctx.Value("user_id")
	userID := strUserID.(int)

	requestEntity := model.URLCreateRequest{
		OriginalURL: req.GetUrl(),
		CreatedBy:   userID,
	}
	shortURL, err := s.URLService.Create(ctx, requestEntity, s.Cfg.ResponseDomain)
	if err != nil {
		return nil, status.Errorf(codes.AlreadyExists, "failed to create shorten URL: %s", err)
	}

	response.SetResult(shortURL)
	return &response, nil
}

func (s *GRPCServerHandler) ListUserURLs(ctx context.Context, empty *emptypb.Empty) (*pb.UserURLsResponse, error) {
	var response pb.UserURLsResponse

	strUserID := ctx.Value("user_id")
	userID := strUserID.(int)

	x, err := s.URLService.GetUserList(ctx, s.Cfg.ResponseDomain, userID)
	if err != nil {
		return nil, status.Errorf(codes.Aborted, "failed to list user URLs: %s", err)
	}
	var y []*pb.URLData
	for _, url := range x {
		i := pb.URLData{}
		i.SetShortUrl(url.OriginalURL)
		i.SetOriginalUrl(url.OriginalURL)
		y = append(y, &i)
	}
	response.SetUrl(y)
	return &response, nil
}

func (s *GRPCServerHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	response := &pb.LoginResponse{}
	peers, ok := peer.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to extract peer from ctx")
	}
	userID, err := s.UserService.GetByIP(ctx, peers.Addr.String())
	if userID != -1 && err == nil {
		jwt, err := s.UserService.Login(userID, s.Cfg)
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
	jwt, err := s.UserService.Login(user.ID, s.Cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	response.SetToken(jwt)
	return response, nil
}
