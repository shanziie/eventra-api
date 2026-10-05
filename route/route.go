package route

import (
	"context"
	"time"

	"eventra-api/helper"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Setup mendaftarkan semua route ke aplikasi.
// Global middleware sudah dipasang di config.NewApp sebelum fungsi ini dipanggil.
func Setup(app *fiber.App, db *pgxpool.Pool) {
	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		// Timeout lebih pendek dari operasi DB biasa karena health check harus cepat.
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("database tidak dapat dijangkau")
		}
		return helper.Success(c, "server berjalan normal", fiber.Map{"status": "ok"})
	})

	// Tangkap semua path yang tidak terdaftar — harus di paling bawah.
	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})
}
