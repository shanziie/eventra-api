package middleware

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"eventra-api/helper"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// Register memasang semua middleware global ke aplikasi Fiber.
// Urutan penting: requestid harus pertama, RequestLogger sebelum recover
// agar bisa mencatat status final setelah recover menangani panic.
func Register(app *fiber.App, allowedOrigins string, logger *slog.Logger) {
	app.Use(requestid.New())
	app.Use(requestLogger(logger))
	app.Use(recover.New(recover.Config{EnableStackTrace: false}))
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{AllowOrigins: allowedOrigins}))
}

// RequireJSON mengembalikan 415 bila Content-Type bukan application/json
// pada method yang membawa body. Dipasang per grup route, bukan global.
func RequireJSON(c *fiber.Ctx) error {
	method := c.Method()
	if method == fiber.MethodPost || method == fiber.MethodPut || method == fiber.MethodPatch {
		ct := c.Get("Content-Type")
		if !strings.HasPrefix(ct, "application/json") {
			return helper.UnsupportedMediaType("content-type harus application/json")
		}
	}
	return c.Next()
}

// requestLogger adalah access log satu baris per request.
// Status final dihitung dari error yang dikembalikan bila ErrorHandler belum berjalan.
func requestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		chainErr := c.Next()

		// Tentukan status final: dari error (ErrorHandler belum berjalan) atau dari response.
		status := c.Response().StatusCode()
		if chainErr != nil {
			var appErr *helper.AppError
			if errors.As(chainErr, &appErr) {
				status = appErr.Status
			} else {
				status = fiber.StatusInternalServerError
			}
		}

		// user_id dan role diset oleh RequireAuth; kosong di endpoint publik.
		userID, _ := c.Locals("user_id").(int)
		role, _ := c.Locals("role").(string)

		attrs := []any{
			"request_id", c.GetRespHeader("X-Request-Id"),
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"ip", c.IP(),
			"user_id", userID,
			"role", role,
		}

		switch {
		case status >= 500:
			logger.Log(context.Background(), slog.LevelError, "request_failed", attrs...)
		case status >= 400:
			logger.Log(context.Background(), slog.LevelWarn, "request_rejected", attrs...)
		default:
			logger.Log(context.Background(), slog.LevelInfo, "request_completed", attrs...)
		}

		return chainErr
	}
}
