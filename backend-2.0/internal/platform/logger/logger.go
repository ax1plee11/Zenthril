package logger

import (
	"log/slog"
	"os"
)

func New(environment string) *slog.Logger {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     slog.LevelInfo,
		AddSource: environment == "development",
	}
	if environment == "development" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}
