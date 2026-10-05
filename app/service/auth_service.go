package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"eventra-api/app/model"
	"eventra-api/app/repository"
	"eventra-api/helper"

	"github.com/gofiber/fiber/v2"
)

// AuthService menangani logika autentikasi dan pendaftaran pengguna.
type AuthService struct {
	userRepo       repository.UserRepository
	tokenRepo      repository.TokenRepository
	jwtManager     *helper.JWTManager
	refreshTTLDays int
}

func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	jwtManager *helper.JWTManager,
	refreshTTLDays int,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		tokenRepo:      tokenRepo,
		jwtManager:     jwtManager,
		refreshTTLDays: refreshTTLDays,
	}
}

// Register mendaftarkan pengguna baru dengan role default participant.
func (s *AuthService) Register(c *fiber.Ctx) error {
	var req model.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	// Hash password dengan bcrypt cost 12
	hashedPassword, err := helper.HashPassword(req.Password)
	if err != nil {
		return helper.Internal(err)
	}

	// Sesuai BR-U1: role selalu ditentukan oleh server sebagai participant
	newUser := &model.User{
		Username: req.Username,
		Email:    req.Email,
		FullName: req.FullName,
		Password: hashedPassword,
		Role:     "participant",
		IsActive: true,
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	created, err := s.userRepo.Create(ctx, newUser)
	if err != nil {
		var dupErr *repository.DuplicateError
		if errors.As(err, &dupErr) {
			cons := strings.ToLower(dupErr.Constraint)
			if strings.Contains(cons, "username") {
				return helper.Conflict("username sudah digunakan")
			}
			if strings.Contains(cons, "email") {
				return helper.Conflict("email sudah digunakan")
			}
			return helper.Conflict("username atau email sudah terdaftar")
		}
		return helper.Internal(err)
	}

	authUser := model.AuthUser{
		ID:        created.ID,
		Username:  created.Username,
		Email:     created.Email,
		FullName:  created.FullName,
		Role:      created.Role,
		IsActive:  created.IsActive,
		CreatedAt: created.CreatedAt,
	}

	location := fmt.Sprintf("/api/v1/users/%d", created.ID)
	return helper.Created(c, location, "registrasi berhasil", authUser)
}

// Login mengautentikasi pengguna dan mengembalikan pasangan access serta refresh token.
func (s *AuthService) Login(c *fiber.Ctx) error {
	var req model.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			// Jalankan dummy hash agar waktu respons sama persis dan menutup timing attack
			helper.DummyHash(req.Password)
			return helper.Unauthorized("username atau password salah")
		}
		return helper.Internal(err)
	}

	if !helper.CheckPassword(user.Password, req.Password) {
		return helper.Unauthorized("username atau password salah")
	}

	// Sesuai BR-U3: user nonaktif tidak bisa login dan seluruh refresh token dicabut
	if !user.IsActive {
		_ = s.tokenRepo.RevokeAllForUser(ctx, user.ID)
		return helper.Forbidden("akun dinonaktifkan, hubungi administrator")
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return helper.Internal(err)
	}

	// Buat refresh token acak 32 byte yang disimpan sebagai SHA-256
	rawRefreshToken, err := helper.RandomToken(32)
	if err != nil {
		return helper.Internal(err)
	}

	tokenHash := helper.SHA256Hex(rawRefreshToken)
	expiresAt := time.Now().Add(time.Duration(s.refreshTTLDays) * 24 * time.Hour)

	dbToken := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
	}

	if err := s.tokenRepo.Save(ctx, dbToken); err != nil {
		return helper.Internal(err)
	}

	resp := model.LoginResponse{
		User: model.AuthUser{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			FullName:  user.FullName,
			Role:      user.Role,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
		},
		Tokens: model.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: rawRefreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    15 * 60,
		},
	}

	return helper.Success(c, "login berhasil", resp)
}

// Refresh merotasi refresh token lama menjadi pasangan token baru.
func (s *AuthService) Refresh(c *fiber.Ctx) error {
	var req model.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	tokenHash := helper.SHA256Hex(req.RefreshToken)

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	activeToken, err := s.tokenRepo.FindActive(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.Unauthorized("refresh token tidak valid atau telah kedaluwarsa")
		}
		return helper.Internal(err)
	}

	// Rotasi token: cabut token lama segera sebelum menerbitkan token baru
	if err := s.tokenRepo.Revoke(ctx, tokenHash); err != nil {
		return helper.Internal(err)
	}

	user, err := s.userRepo.FindByID(ctx, activeToken.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.Unauthorized("pengguna tidak ditemukan")
		}
		return helper.Internal(err)
	}

	if !user.IsActive {
		return helper.Forbidden("akun dinonaktifkan")
	}

	newAccessToken, err := s.jwtManager.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return helper.Internal(err)
	}

	newRawRefreshToken, err := helper.RandomToken(32)
	if err != nil {
		return helper.Internal(err)
	}

	newTokenHash := helper.SHA256Hex(newRawRefreshToken)
	expiresAt := time.Now().Add(time.Duration(s.refreshTTLDays) * 24 * time.Hour)

	newDBToken := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: newTokenHash,
		ExpiresAt: expiresAt,
	}

	if err := s.tokenRepo.Save(ctx, newDBToken); err != nil {
		return helper.Internal(err)
	}

	resp := model.LoginResponse{
		User: model.AuthUser{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			FullName:  user.FullName,
			Role:      user.Role,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
		},
		Tokens: model.TokenPair{
			AccessToken:  newAccessToken,
			RefreshToken: newRawRefreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    15 * 60,
		},
	}

	return helper.Success(c, "refresh token berhasil", resp)
}

// Logout mencabut refresh token agar tidak bisa digunakan kembali.
func (s *AuthService) Logout(c *fiber.Ctx) error {
	var req model.LogoutRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("format body json tidak valid")
	}

	if err := helper.ValidateStruct(&req); err != nil {
		return err
	}

	tokenHash := helper.SHA256Hex(req.RefreshToken)

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	_ = s.tokenRepo.Revoke(ctx, tokenHash)
	return helper.Success(c, "logout berhasil", nil)
}

// Me mengembalikan data profil pengguna saat ini beserta daftar hak akses permission miliknya.
func (s *AuthService) Me(c *fiber.Ctx) error {
	userID, err := helper.CurrentUserID(c)
	if err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return helper.NotFound("pengguna tidak ditemukan")
		}
		return helper.Internal(err)
	}

	permissions := helper.GetPermissionsByRole(user.Role)

	resp := model.MeResponse{
		User: model.AuthUser{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			FullName:  user.FullName,
			Role:      user.Role,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
		},
		Permissions: permissions,
	}

	return helper.Success(c, "data profil berhasil diambil", resp)
}
