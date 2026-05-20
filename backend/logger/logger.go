package logger

import (
	"log/slog"
	"os"
)

var L *slog.Logger

func Init(env string) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: env == "production",
	}

	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	L = slog.New(handler)
	slog.SetDefault(L)
}

func Info(msg string, args ...any) {
	L.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	L.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	L.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	L.Debug(msg, args...)
}
