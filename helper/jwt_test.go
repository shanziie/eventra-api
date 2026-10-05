package helper_test

import (
	"errors"
	"testing"
	"time"

	"eventra-api/helper"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTManager_GenerateAndVerify(t *testing.T) {
	secret := "super-secret-key-that-is-at-least-32-chars-long"
	issuer := "eventra-test"
	manager := helper.NewJWTManager(secret, issuer, 15)

	token, err := manager.GenerateAccessToken(42, "organizer")
	if err != nil {
		t.Fatalf("gagal membuat access token: %v", err)
	}

	claims, err := manager.VerifyToken(token)
	if err != nil {
		t.Fatalf("gagal memverifikasi token valid: %v", err)
	}

	if claims.UserID != 42 {
		t.Errorf("expected UserID 42, got %d", claims.UserID)
	}
	if claims.Role != "organizer" {
		t.Errorf("expected Role organizer, got %s", claims.Role)
	}
	if claims.Issuer != issuer {
		t.Errorf("expected Issuer %s, got %s", issuer, claims.Issuer)
	}
}

func TestJWTManager_ExpiredToken(t *testing.T) {
	secret := "super-secret-key-that-is-at-least-32-chars-long"
	issuer := "eventra-test"

	// Buat token yang sudah kedaluwarsa secara manual
	claims := helper.JWTClaims{
		UserID: 1,
		Role:   "participant",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := tokenObj.SignedString([]byte(secret))

	manager := helper.NewJWTManager(secret, issuer, 15)
	_, err := manager.VerifyToken(tokenStr)
	if err == nil {
		t.Fatal("diharapkan error expired token, namun nil")
	}
	if !errors.Is(err, helper.ErrExpiredToken) {
		t.Errorf("expected ErrExpiredToken, got %v", err)
	}
}

func TestJWTManager_DifferentSecret(t *testing.T) {
	secretA := "super-secret-key-that-is-at-least-32-chars-long-A"
	secretB := "super-secret-key-that-is-at-least-32-chars-long-B"
	issuer := "eventra-test"

	managerA := helper.NewJWTManager(secretA, issuer, 15)
	managerB := helper.NewJWTManager(secretB, issuer, 15)

	token, err := managerA.GenerateAccessToken(10, "admin")
	if err != nil {
		t.Fatalf("gagal generate: %v", err)
	}

	_, err = managerB.VerifyToken(token)
	if err == nil {
		t.Fatal("verifikasi harus gagal bila secret berbeda")
	}
	if !errors.Is(err, helper.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWTManager_AlgNoneRejected(t *testing.T) {
	issuer := "eventra-test"
	manager := helper.NewJWTManager("super-secret-key-that-is-at-least-32-chars-long", issuer, 15)

	// Buat token dengan alg: "none" (SigningMethodNone)
	claims := helper.JWTClaims{
		UserID: 1,
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "1",
			Issuer:    issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	noneToken, err := tokenObj.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("gagal membuat none token: %v", err)
	}

	_, err = manager.VerifyToken(noneToken)
	if err == nil {
		t.Fatal("token dengan alg 'none' harus ditolak")
	}
	if !errors.Is(err, helper.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestJWTManager_TamperedToken(t *testing.T) {
	manager := helper.NewJWTManager("super-secret-key-that-is-at-least-32-chars-long", "eventra-test", 15)
	token, _ := manager.GenerateAccessToken(5, "participant")

	// Ubah satu karakter di akhir token signature
	tampered := token[:len(token)-2] + "X" + token[len(token)-1:]
	_, err := manager.VerifyToken(tampered)
	if err == nil {
		t.Fatal("token yang diubah harus ditolak")
	}
	if !errors.Is(err, helper.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}
