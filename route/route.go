package route

import (
	"context"
	"time"

	"eventra-api/app/service"
	"eventra-api/helper"
	"eventra-api/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Setup mendaftarkan semua route ke aplikasi Fiber.
func Setup(
	app *fiber.App,
	db *pgxpool.Pool,
	authService *service.AuthService,
	jwtManager *helper.JWTManager,
) {
	api := app.Group("/api/v1")

	// Endpoint health check
	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("database tidak dapat dijangkau")
		}
		return helper.Success(c, "server berjalan normal", fiber.Map{"status": "ok"})
	})

	// Grup Auth (Modul 5, PRD 8.2 endpoint 2 - 6)
	auth := api.Group("/auth")
	auth.Post("/register", middleware.RequireJSON, authService.Register)
	auth.Post("/login", middleware.RequireJSON, middleware.LoginRateLimiter(), authService.Login)
	auth.Post("/refresh", middleware.RequireJSON, authService.Refresh)
	auth.Post("/logout", middleware.RequireJSON, authService.Logout)
	auth.Get("/me", middleware.RequireAuth(jwtManager), authService.Me)

	// Tangkap semua route yang tidak terdaftar — harus di posisi paling akhir
	app.Use(func(c *fiber.Ctx) error {
		return helper.NotFound("endpoint tidak ditemukan")
	})
}
