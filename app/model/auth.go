package model

import "time"

// RegisterRequest memuat data pendaftaran akun baru.
// Sesuai BR-U1, tidak ada field role di struct ini (server selalu menetapkan role participant).
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30,username"`
	Email    string `json:"email" validate:"required,email,max=120"`
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	Password string `json:"password" validate:"required,max=72,strongpassword"`
}

// LoginRequest memuat kredensial masuk pengguna.
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest memuat refresh token untuk rotasi token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest memuat refresh token yang akan dicabut.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// TokenPair adalah pasangan access token dan refresh token yang dikembalikan saat login/refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // detik
}

// LoginResponse adalah payload response saat login atau refresh token berhasil.
type LoginResponse struct {
	User   AuthUser  `json:"user"`
	Tokens TokenPair `json:"tokens"`
}

// RefreshToken merepresentasikan data token penyegar di database.
type RefreshToken struct {
	ID        int64      `json:"id"`
	UserID    int        `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// MeResponse adalah response untuk endpoint GET /auth/me.
type MeResponse struct {
	User        AuthUser `json:"user"`
	Permissions []string `json:"permissions"`
}
