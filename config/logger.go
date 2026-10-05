package config

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// InitLogger menyiapkan logger slog JSON yang menulis ke stdout dan logs/app.log.
// Rotasi file diatur oleh lumberjack: 10 MB per file, 5 backup, retensi 14 hari.
func InitLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	fileWriter := &lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    10, // MB
		MaxBackups: 5,
		MaxAge:     14, // hari
		Compress:   true,
	}

	// Tulis ke stdout (untuk docker/terminal) dan file sekaligus.
	multi := io.MultiWriter(os.Stdout, fileWriter)

	logger := slog.New(slog.NewJSONHandler(multi, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(logger)
	return logger
}
