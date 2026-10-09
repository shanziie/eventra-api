package model

import "time"

// data pendaftaran akun baru.
// tidak ada field role di struct ini (server selalu menetapkan role participant)
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30,username"`
	Email    string `json:"email" validate:"required,email,max=120"`
	FullName string `json:"full_name" validate:"required,min=2,max=100"`
	Password string `json:"password" validate:"required,max=72,strongpassword"`
}

// kredensial masuk pengguna
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// refresh token untuk rotasi token
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// refresh token yang akan dicabut
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// pasangan access token dan refresh token yang dikembalikan saat login/refresh
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // detik
}

// payload response saat login atau refresh token berhasil
type LoginResponse struct {
	User   AuthUser  `json:"user"`
	Tokens TokenPair `json:"tokens"`
}

// data token penyegar di database
type RefreshToken struct {
	ID        int64      `json:"id"`
	UserID    int        `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// response untuk endpoint GET /auth/me
type MeResponse struct {
	User        AuthUser `json:"user"`
	Permissions []string `json:"permissions"`
}
