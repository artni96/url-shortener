package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/auth"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

var ErrWrongUsernameOrPassword = errors.New("wrong username or password")
var ErrWrongPassword = errors.New("passwords do not match")

type AuthServiceInterface interface {
	Create(ctx context.Context, user model.UserCreateRequest) error
	Login(ctx context.Context, user model.UserLogin) (string, error)
}
type AuthService struct {
	dbRepository auth.DBAuthRepositoryInterface
	app          *config.App
}

func (s *AuthService) Create(ctx context.Context, user model.UserCreateRequest) error {
	if user.Password != user.RepeatedPassword {
		return ErrWrongPassword
	}
	encryptedPassword, err := s.EncryptPassword(user.Password)
	if err != nil {
		s.app.Logger.Info("failed to encrypt password", zap.Error(err))
		return err
	}

	entityToCreate := model.UserCreate{
		Username:       user.Username,
		HashedPassword: encryptedPassword,
	}

	_, err = s.dbRepository.Create(ctx, entityToCreate)
	if err != nil {
		s.app.Logger.Info("failed to create user", zap.Error(err), zap.String("username", user.Username))
		return err
	}
	s.app.Logger.Info("user successfully created", zap.String("user", entityToCreate.Username))
	return nil
}

func (s *AuthService) Login(ctx context.Context, user model.UserLogin) (string, error) {
	encryptedPassword, err := s.EncryptPassword(user.Password)
	if err != nil {
		s.app.Logger.Info("failed to encrypt password", zap.Error(err))
		return "", err
	}
	dbUserEntity, err := s.dbRepository.GetUserHashedPassword(ctx, user)
	if err != nil {
		s.app.Logger.Info("failed to get user hashed password", zap.Error(err))
		return "", err
	}

	if dbUserEntity.Password != encryptedPassword {
		return "", ErrWrongUsernameOrPassword
	}

	token, err := s.BuildJWTString(dbUserEntity.ID)
	if err != nil {
		s.app.Logger.Info("failed to build token", zap.Error(err))
		return "", err
	}
	s.app.Logger.Debug("user successfully authenticated", zap.String("user", user.Username))
	return token, nil
}

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const (
	TOKEN_EXP  = time.Hour * 3
	SECRET_KEY = "dontshareme"
)

func (s *AuthService) EncryptPassword(password string) (string, error) {
	key := sha256.Sum256([]byte(SECRET_KEY))
	aesblock, err := aes.NewCipher(key[:])
	if err != nil {
		return "", fmt.Errorf("failed to create aes block: %v", err)
	}
	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return "", fmt.Errorf("failed to create aes gcm: %v", err)
	}
	nonce := make([]byte, aesgcm.NonceSize())

	encrypted := aesgcm.Seal(nonce, nonce, []byte(password), nil)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func (s *AuthService) BuildJWTString(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: userID,
	})
	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %v", err)
	}
	return tokenString, nil
}

func NewAuthService(dbRepository auth.DBAuthRepositoryInterface, app *config.App) *AuthService {
	return &AuthService{
		dbRepository: dbRepository,
		app:          app,
	}
}
