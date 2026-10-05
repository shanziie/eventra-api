package helper_test

import (
	"strings"
	"testing"

	"eventra-api/helper"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndCheckPassword(t *testing.T) {
	password := "Rahasia#2026"

	hash, err := helper.HashPassword(password)
	if err != nil {
		t.Fatalf("gagal hash password: %v", err)
	}

	// Pastikan diawali $2a$12$ sesuai spesifikasi bcrypt cost 12
	if !strings.HasPrefix(hash, "$2a$12$") {
		t.Errorf("hash harus diawali $2a$12$, got: %s", hash[:7])
	}

	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("gagal membaca cost hash: %v", err)
	}
	if cost != helper.BcryptCost {
		t.Errorf("expected bcrypt cost %d, got %d", helper.BcryptCost, cost)
	}

	if !helper.CheckPassword(hash, password) {
		t.Error("CheckPassword harus bernilai true untuk password yang benar")
	}

	if helper.CheckPassword(hash, "PasswordSalah!99") {
		t.Error("CheckPassword harus bernilai false untuk password yang salah")
	}
}

func TestDummyHash(t *testing.T) {
	// Memastikan fungsi DummyHash berjalan tanpa panic
	helper.DummyHash("sembarangPassword123")
}

func TestRandomTokenAndSHA256Hex(t *testing.T) {
	token1, err := helper.RandomToken(32)
	if err != nil {
		t.Fatalf("gagal membuat random token: %v", err)
	}
	token2, err := helper.RandomToken(32)
	if err != nil {
		t.Fatalf("gagal membuat random token: %v", err)
	}

	if token1 == token2 {
		t.Error("token acak harus unik satu sama lain")
	}

	// 32 byte dalam hex menghasilkan 64 karakter
	if len(token1) != 64 {
		t.Errorf("expected panjang hex 64, got %d", len(token1))
	}

	hash1 := helper.SHA256Hex(token1)
	hash2 := helper.SHA256Hex(token1)
	if hash1 != hash2 {
		t.Error("SHA256Hex harus deterministik untuk input yang sama")
	}

	if len(hash1) != 64 {
		t.Errorf("expected panjang sha256 hex 64, got %d", len(hash1))
	}
}
