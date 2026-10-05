package helper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost diatur ke 12 sesuai spesifikasi AGENTS.md §2 dan §6.
const BcryptCost = 12

// dummyBcryptHash dibuat dengan cost 12 untuk mencegah timing attack saat username tidak ditemukan.
const dummyBcryptHash = "$2a$12$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

// HashPassword menghasilkan hash bcrypt cost 12 dari plain password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPassword memverifikasi kecocokan antara plain password dan hash bcrypt.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// DummyHash memproses password terhadap hash dummy untuk menyamakan waktu komputasi
// saat user tidak ditemukan, menutup celah user enumeration via timing attack.
func DummyHash(password string) {
	_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
}

// RandomToken menghasilkan string acak aman sepanjang n byte dalam format hexadecimal.
func RandomToken(bytesLen int) (string, error) {
	b := make([]byte, bytesLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// SHA256Hex menghitung hash SHA-256 dari string input dan mengembalikan representasi heksadesimalnya.
func SHA256Hex(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])
}
