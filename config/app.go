package config

import (
	"errors"
	"log/slog"

	"eventra-api/helper"
	"eventra-api/middleware"

	"github.com/gofiber/fiber/v2"
)

// NewApp membuat instance Fiber dengan konfigurasi global: BodyLimit, ErrorHandler terpusat,
// dan semua middleware global.
func NewApp(cfg *Config, logger *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		// Batasi ukuran body untuk mencegah request besar yang menyebabkan OOM.
		BodyLimit:    1 * 1024 * 1024,
		ErrorHandler: makeErrorHandler(logger),
	})

	middleware.Register(app, cfg.AllowedOrigins, logger)
	return app
}

// makeErrorHandler mengembalikan ErrorHandler terpusat.
// Semua handler yang gagal HARUS mengembalikan *helper.AppError.
// ErrorHandler ini satu-satunya yang menulis response gagal.
// cause tidak diakses langsung (field tidak diekspor); err.Error() sudah menyertakannya untuk log.
func makeErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		var appErr *helper.AppError
		if !errors.As(err, &appErr) {
			// Error yang tidak dikenal (mis. dari recover) diterjemahkan ke 500.
			appErr = helper.Internal(err)
		}

		requestID := c.GetRespHeader("X-Request-Id")

		// Log sudah dilakukan oleh requestLogger; di sini hanya mencatat detail error
		// bila status 5xx agar mudah ditelusuri dari request_id.
		if appErr.Status >= 500 {
			logger.Error("internal error detail", "error", err.Error(), "request_id", requestID)
		}

		// fields sengaja diperiksa agar tidak muncul "fields:null" di JSON response.
		resp := fiber.Map{
			"success":    false,
			"code":       appErr.Code,
			"message":    appErr.Message,
			"request_id": requestID,
		}
		if len(appErr.Fields) > 0 {
			resp["fields"] = appErr.Fields
		}

		return c.Status(appErr.Status).JSON(resp)
	}
}
