package middleware

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"eventra-api/helper"

	"github.com/gofiber/fiber/v2"
)

// RequireAuth memverifikasi bearer access token dan menyematkan user_id serta role ke context.
// Jika gagal, mengembalikan status 401 beserta header WWW-Authenticate.
func RequireAuth(jwtManager *helper.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Set("WWW-Authenticate", `Bearer realm="eventra"`)
			return helper.Unauthorized("token otorisasi diperlukan")
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtManager.VerifyToken(tokenString)
		if err != nil {
			c.Set("WWW-Authenticate", `Bearer realm="eventra", error="invalid_token"`)
			return helper.Unauthorized(err.Error())
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("role", claims.Role)
		return c.Next()
	}
}

type loginAttemptTracker struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

var loginTracker = &loginAttemptTracker{
	attempts: make(map[string][]time.Time),
}

// LoginRateLimiter membatasi percobaan login maksimal 5 kali per menit per IP address.
// Bila terlampaui, mengembalikan status 429 dan header Retry-After.
func LoginRateLimiter() fiber.Handler {
	const maxAttempts = 5
	const window = 1 * time.Minute

	return func(c *fiber.Ctx) error {
		ip := c.IP()
		now := time.Now()

		loginTracker.mu.Lock()
		defer loginTracker.mu.Unlock()

		// Bersihkan catatan di luar jendela waktu 1 menit
		validAttempts := make([]time.Time, 0, maxAttempts)
		for _, t := range loginTracker.attempts[ip] {
			if now.Sub(t) < window {
				validAttempts = append(validAttempts, t)
			}
		}

		if len(validAttempts) >= maxAttempts {
			oldest := validAttempts[0]
			retryAfter := int(window.Seconds() - now.Sub(oldest).Seconds())
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Set("Retry-After", strconv.Itoa(retryAfter))
			loginTracker.attempts[ip] = validAttempts
			return helper.TooManyRequests("terlalu banyak percobaan login, silakan coba lagi beberapa saat")
		}

		validAttempts = append(validAttempts, now)
		loginTracker.attempts[ip] = validAttempts
		return c.Next()
	}
}
