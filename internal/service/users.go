package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/users"
)

type UserServiceInterface interface {
	Create(ctx context.Context, ip string) (model.User, error)
	Login(userID int, cfg *config.Config) (string, error)
	GetByIP(ctx context.Context, ip string) (int, error)

	BuildJWTString(userID int, cfg *config.Config) (string, error)
}
type UserService struct {
	dbRepository       users.DBUserRepositoryInterface
	inMemoryRepository users.InMemoryUserRepositoryInterface
	app                *config.App
}

func (s *UserService) Create(ctx context.Context, ip string) (model.User, error) {
	var responseUser model.User
	var err error
	var user model.User
	if s.dbRepository != nil {
		user, err = s.dbRepository.Create(ctx, ip)
	} else {
		user, err = s.inMemoryRepository.Create(ip)
	}
	if err != nil {
		s.app.Logger.Info("failed to create user", zap.Error(err))
		responseUser.ID = -1
		return responseUser, err
	}
	if s.dbRepository == nil && s.app.Cfg.FileStoragePath != "" {
		fileWriter, err := users.NewWriter(s.app.Cfg.FileStoragePath)
		if err != nil {
			s.app.Logger.Error("could not create file writer",
				zap.String("path", s.app.Cfg.FileStoragePath),
				zap.String("error message", err.Error()),
			)
			responseUser.ID = -1
			return responseUser, err
		}
		defer fileWriter.Close()

		err = fileWriter.WriteEntity(user)
		if err != nil {
			responseUser.ID = -1
			return responseUser, fmt.Errorf("could not write entity to file: %w", err)
		}
	}
	s.app.Logger.Debug("user successfully created", zap.Int("user id", user.ID))
	responseUser.ID = user.ID
	responseUser.IP = ip
	return responseUser, nil
}

func (s *UserService) Login(userID int, cfg *config.Config) (string, error) {

	token, err := s.BuildJWTString(userID, cfg)
	if err != nil {
		s.app.Logger.Info("failed to build token", zap.Error(err))
		return "", err
	}
	s.app.Logger.Debug("user successfully authenticated", zap.Int("userID", userID))
	return token, nil
}

func (s *UserService) GetByIP(ctx context.Context, ip string) (int, error) {
	var userID int
	var err error
	if s.dbRepository != nil {
		userID, err = s.dbRepository.GetByIP(ctx, ip)
	} else {
		userID, err = s.inMemoryRepository.GetByIP(ip)
	}

	if err != nil {
		s.app.Logger.Info("failed to get user by ip", zap.String("ip", ip), zap.Error(err))
		return -1, err
	}
	s.app.Logger.Debug("user successfully retrieved", zap.Int("userID", userID))
	return userID, nil
}

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

func (s *UserService) BuildJWTString(userID int, cfg *config.Config) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.TokenExp)),
		},
		UserID: userID,
	})
	tokenString, err := token.SignedString([]byte(cfg.SecretKey))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %v", err)
	}
	return tokenString, nil
}

func GetUserID(tokenString string, cfg *config.Config) int {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(cfg.SecretKey), nil
	})
	if err != nil {
		return -1
	}

	if !token.Valid {
		return -1
	}
	return claims.UserID
}

func NewUserService(dbRepository users.DBUserRepositoryInterface, inMemoryRepository users.InMemoryUserRepositoryInterface, app *config.App) *UserService {
	return &UserService{
		dbRepository:       dbRepository,
		inMemoryRepository: inMemoryRepository,
		app:                app,
	}
}
