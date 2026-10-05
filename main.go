package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eventra-api/app/repository"
	"eventra-api/app/service"
	"eventra-api/config"
	"eventra-api/database"
	"eventra-api/helper"
	"eventra-api/route"
)

func main() {
	cfg := config.Load()
	logger := config.InitLogger(cfg.LogLevel)

	// Validasi keamanan: JWT_SECRET minimal 32 karakter untuk algoritma HS256
	if len(cfg.JWTSecret) < 32 {
		logger.Error("JWT_SECRET harus minimal 32 karakter, aplikasi dihentikan")
		os.Exit(1)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		logger.Error("gagal koneksi database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Muat seluruh tabel RBAC sekali saat startup sesuai AGENTS.md §6 (fail closed)
	initCtx, initCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := helper.LoadPermissions(initCtx, db); err != nil {
		initCancel()
		logger.Error("gagal memuat permission RBAC dari database", "error", err)
		os.Exit(1)
	}
	initCancel()

	// Inisialisasi dependency layer (Repository -> Service)
	userRepo := repository.NewUserRepository(db)
	tokenRepo := repository.NewTokenRepository(db)
	jwtManager := helper.NewJWTManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAccessTTL)
	authService := service.NewAuthService(userRepo, tokenRepo, jwtManager, cfg.JWTRefreshTTLDays)

	app := config.NewApp(cfg, logger)
	route.Setup(app, db, authService, jwtManager)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := ":" + cfg.AppPort
		logger.Info("server dimulai", "name", cfg.AppName, "addr", addr)
		if err := app.Listen(addr); err != nil {
			slog.Error("server berhenti tidak normal", "error", err)
		}
	}()

	<-quit
	logger.Info("menerima sinyal shutdown, menyelesaikan request yang berjalan...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Error("gagal graceful shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("server berhenti dengan bersih")
}
