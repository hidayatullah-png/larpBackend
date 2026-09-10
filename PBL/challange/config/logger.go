package config

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Newlogger membuat logger terstruktur yang menulis ke dua tujuan: berkas rotasi dan konsol.
// layar (stdout) agar terlihat saat praktikum, dan file logs/app.log yang berotasi agar tidak membengkak.
func NewLogger() *slog.Logger {
	if err := os.MkdirAll("logs", 0755); err != nil {
		panic("gagal membuat direktori logs: " + err.Error())
	}

	rotator := &lumberjack.Logger{
		Filename:   filepath.Join("logs", "app.log"),
		MaxSize:    10, // megabytes
		MaxBackups: 5,
		MaxAge:     14,   //days
		Compress:   true, // disabled by default
	}

	writer := io.MultiWriter(os.Stdout, rotator)
	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: parseLevel(GetEnv("LOG_LEVEL", "INFO")),
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func parseLevel(value string) slog.Level {
	switch strings.ToUpper(value) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
