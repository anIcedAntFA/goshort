package config

import (
	"log/slog"
	"os"
)

// SetupLogger configures the global slog logger based on LoggingConfig.
func SetupLogger(cfg LoggingConfig) {
	logLevel := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: logLevel}
	var handler slog.Handler
	if cfg.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}
