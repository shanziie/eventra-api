package helper

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("token tidak valid")
	ErrExpiredToken = errors.New("token telah kedaluwarsa")
)

// JWTClaims memuat klaim standar dan kustom untuk access token.
type JWTClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTManager mengelola pembuatan dan verifikasi JWT token.
type JWTManager struct {
	secret       []byte
	issuer       string
	accessTTLMin int
}

func NewJWTManager(secret string, issuer string, accessTTLMin int) *JWTManager {
	return &JWTManager{
		secret:       []byte(secret),
		issuer:       issuer,
		accessTTLMin: accessTTLMin,
	}
}

// GenerateAccessToken membuat token JWT HS256 dengan claims sub, role, iss, iat, dan exp.
func (m *JWTManager) GenerateAccessToken(userID int, role string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(m.accessTTLMin) * time.Minute)

	claims := JWTClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(userID),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("jwt sign: %w", err)
	}
	return signed, nil
}

// VerifyToken memvalidasi token JWT secara ketat: metode harus HMAC eksplisit, issuer cocok, belum kedaluwarsa.
func (m *JWTManager) VerifyToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(t *jwt.Token) (any, error) {
		// Verifikasi algoritma HMAC eksplisit untuk mencegah serangan token alg:none atau algoritma asimetris palsu
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("algoritma tidak valid: %v", t.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer))

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
