package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/model"
	"github.com/artni96/url-shortener/internal/repository/auth"
	"github.com/golang-jwt/jwt/v4"
)

var ErrWrongUsernameOrPassword = errors.New("wrong username or password")

type AuthServiceInterface interface {
	Create(ctx context.Context, user model.UserCreateRequest) (string, error)
	Login(ctx context.Context, user model.UserLogin) (string, error)
}
type AuthService struct {
	dbRepository auth.DBAuthRepositoryInterface
	app          *config.App
}

func (s *AuthService) Create(ctx context.Context, user model.UserCreateRequest) (string, error) {
	encryptedPassword, err := EncryptPassword(user.Password)
	if err != nil {
		return "", err
	}

	entityToCreate := model.UserCreate{
		Username:       user.Username,
		HashedPassword: encryptedPassword,
	}

	result, err := s.dbRepository.Create(ctx, entityToCreate)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (s *AuthService) Login(ctx context.Context, user model.UserLogin) (string, error) {
	encryptedPassword, err := EncryptPassword(user.Password)
	if err != nil {
		return "", err
	}
	dbUserPassword, err := s.dbRepository.GetUserHashedPassword(ctx, user)
	if err != nil {
		return "", err
	}

	if dbUserPassword != encryptedPassword {
		return "", ErrWrongUsernameOrPassword
	}
	return "", nil
}

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const (
	TOKEN_EXP  = time.Hour * 3
	SECRET_KEY = "dontsharemepleasedontsharemeplea"
)

func EncryptPassword(password string) (string, error) {
	key := []byte(SECRET_KEY)
	aesblock, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesgcm.NonceSize())

	encrypted := aesgcm.Seal(nonce, nonce, []byte(password), nil)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func BuildJWTString(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: userID,
	})
	tokenString, err := token.SignedString([]byte(SECRET_KEY))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func NewAuthService(dbRepository auth.DBAuthRepositoryInterface, app *config.App) *AuthService {
	return &AuthService{
		dbRepository: dbRepository,
		app:          app,
	}
}
