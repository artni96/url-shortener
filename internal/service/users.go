package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/artni96/url-shortener/internal/config"
	"github.com/artni96/url-shortener/internal/repository/users"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

var ErrWrongUsernameOrPassword = errors.New("wrong username or password")
var ErrWrongPassword = errors.New("passwords do not match")

type UserServiceInterface interface {
	Create(ctx context.Context) (int, error)
	Login(ctx context.Context, userID int) (string, error)
}
type UserService struct {
	dbRepository       users.DBUserRepositoryInterface
	inMemoryRepository users.InMemoryUserRepositoryInterface
	app                *config.App
}

func (s *UserService) Create(ctx context.Context) (int, error) {
	//if user.Password != user.RepeatedPassword {
	//	return ErrWrongPassword
	//}
	//encryptedPas password, err := s.EncryptPassword(user.Password)
	//if err != nil {
	//	s.app.Logger.Info("failed to encrypt password", zap.Error(err))
	//	return err
	//}

	//entityToCreate := model.UserCreate{
	//	Username:       user.Username,
	//	HashedPassword: encryptedPassword,
	//}
	var userID int
	var err error
	if s.dbRepository != nil {
		userID, err = s.dbRepository.Create(ctx)
	} else {
		userID, err = s.inMemoryRepository.Create()
	}
	if err != nil {
		s.app.Logger.Info("failed to create user", zap.Error(err))
		return -1, err
	}
	if s.dbRepository == nil && s.app.Cfg.FileStoragePath != "" {
		fileWriter, err := users.NewWriter(s.app.Cfg.FileStoragePath)
		if err != nil {
			s.app.Logger.Error("could not create file writer",
				zap.String("path", s.app.Cfg.FileStoragePath),
				zap.String("error message", err.Error()),
			)
			return -1, err
		}
		defer fileWriter.Close()

		err = fileWriter.WriteEntity(userID)
		if err != nil {
			return -1, fmt.Errorf("could not write entity to file: %w", err)
		}
	}
	s.app.Logger.Info("user successfully created", zap.Int("user id", userID))
	return userID, nil
}

func (s *UserService) Login(ctx context.Context, userID int) (string, error) {
	//encryptedPassword, err := s.EncryptPassword(user.Password)
	//if err != nil {
	//	s.app.Logger.Info("failed to encrypt password", zap.Error(err))
	//	return "", err
	//}
	//
	//dbUserEntity := model.UserWithHashedPassword{}
	//if s.dbRepository != nil {
	//	dbUserEntity, err = s.dbRepository.GetUserHashedPassword(ctx, user.Username)
	//} else {
	//	dbUserEntity, err = s.inMemoryRepository.GetUserHashedPassword(user.Username)
	//}

	//if err != nil {
	//	s.app.Logger.Info("failed to get user hashed password", zap.Error(err))
	//	return "", err
	//}
	//
	//if dbUserEntity.Password != encryptedPassword {
	//	return "", ErrWrongUsernameOrPassword
	//}

	token, err := s.BuildJWTString(userID)
	if err != nil {
		s.app.Logger.Info("failed to build token", zap.Error(err))
		return "", err
	}
	s.app.Logger.Debug("user successfully authenticated", zap.Int("userID", userID))
	return token, nil
}

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const (
	TOKEN_EXP  = time.Minute * 1
	SECRET_KEY = "dontshareme"
)

//	func (s *UserService) EncryptPassword(password string) (string, error) {
//		key := sha256.Sum256([]byte(SECRET_KEY))
//		aesblock, err := aes.NewCipher(key[:])
//		if err != nil {
//			return "", fmt.Errorf("failed to create aes block: %v", err)
//		}
//		aesgcm, err := cipher.NewGCM(aesblock)
//		if err != nil {
//			return "", fmt.Errorf("failed to create aes gcm: %v", err)
//		}
//		nonce := make([]byte, aesgcm.NonceSize())
//
//		encrypted := aesgcm.Seal(nonce, nonce, []byte(password), nil)
//		return base64.StdEncoding.EncodeToString(encrypted), nil
//	}
func (s *UserService) BuildJWTString(userID int) (string, error) {
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

func GetUserID(tokenString string) int {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(SECRET_KEY), nil
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
