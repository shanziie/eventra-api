package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"eventra-api/config"
	"eventra-api/database"
	"eventra-api/route"
)

func main() {
	cfg := config.Load()
	logger := config.InitLogger(cfg.LogLevel)

	// JWT_SECRET kurang dari 32 karakter adalah risiko keamanan — hentikan aplikasi.
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

	app := config.NewApp(cfg, logger)
	route.Setup(app, db)

	// Jalankan server di goroutine terpisah agar sinyal shutdown bisa diterima.
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("gagal graceful shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("server berhenti dengan bersih")
}
