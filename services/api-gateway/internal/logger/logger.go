package logger

import (
	"log/slog"
	"os"
)

var Logger *slog.Logger

func init() {
	env := os.Getenv("ENV")

	var level slog.Level
	var handler slog.Handler

	switch env {
	case "production", "prod":
		level = slog.LevelInfo
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	default:
		level = slog.LevelDebug
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}

	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}

func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}

func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}