package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config memuat semua konfigurasi dari environment variable.
type Config struct {
	AppName  string
	AppPort  string
	LogLevel string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBMaxConns int32

	JWTSecret         string
	JWTIssuer         string
	JWTAccessTTL      int
	JWTRefreshTTLDays int

	AllowedOrigins string
}

// Load membaca .env (bila ada) lalu mengembalikan Config dari environment variable.
func Load() *Config {
	// Abaikan error bila .env tidak ada — production menggunakan env asli.
	_ = godotenv.Load()

	return &Config{
		AppName:  getEnv("APP_NAME", "eventra-api"),
		AppPort:  getEnv("APP_PORT", "3000"),
		LogLevel: getEnv("LOG_LEVEL", "info"),

		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "eventra"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		DBMaxConns: int32(getEnvInt("DB_MAX_CONNS", 10)),

		JWTSecret:         getEnv("JWT_SECRET", ""),
		JWTIssuer:         getEnv("JWT_ISSUER", "eventra-api"),
		JWTAccessTTL:      getEnvInt("JWT_ACCESS_TTL_MINUTES", 15),
		JWTRefreshTTLDays: getEnvInt("JWT_REFRESH_TTL_DAYS", 7),

		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "*"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("env %s bukan angka, pakai default %d", key, fallback)
		return fallback
	}
	return n
}
