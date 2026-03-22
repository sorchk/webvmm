package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/webvmm/webvmm/internal/models"
)

var (
	ErrInvalidToken = errors.New("无效的令牌")
	ErrExpiredToken = errors.New("令牌已过期")
)

// Claims JWT 声明
type Claims struct {
	UserID    uint        `json:"user_id"`
	Username  string      `json:"username"`
	Email     string      `json:"email"`
	Role      models.Role `json:"role"`
	TokenType string      `json:"token_type"` // access or refresh
	jwt.RegisteredClaims
}

// JWTService JWT 服务
type JWTService struct {
	secretKey     string
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
}

// NewJWTService 创建 JWT 服务
func NewJWTService(secretKey string, tokenExpiryHours, refreshExpiryHours int) *JWTService {
	return &JWTService{
		secretKey:     secretKey,
		tokenExpiry:   time.Duration(tokenExpiryHours) * time.Hour,
		refreshExpiry: time.Duration(refreshExpiryHours) * time.Hour,
	}
}

// GenerateToken 生成访问令牌
func (s *JWTService) GenerateToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "webvmm",
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

// GenerateRefreshToken 生成刷新令牌
func (s *JWTService) GenerateRefreshToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      user.Role,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "webvmm",
			Subject:   user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secretKey))
}

// ValidateToken 验证令牌
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.secretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshAccessToken 使用刷新令牌生成新的访问令牌
func (s *JWTService) RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return "", err
	}

	if claims.TokenType != "refresh" {
		return "", ErrInvalidToken
	}

	// 创建新的访问令牌声明
	newClaims := &Claims{
		UserID:    claims.UserID,
		Username:  claims.Username,
		Email:     claims.Email,
		Role:      claims.Role,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "webvmm",
			Subject:   claims.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	return token.SignedString([]byte(s.secretKey))
}
