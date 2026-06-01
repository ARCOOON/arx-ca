package logging

import (
	"log/slog"
	"os"
	"strings"
)

var defaultLogger *slog.Logger

// Configure sets the process-wide slog logger from a level name (debug, info, warn, error).
func Configure(level string) {
	lvl := ParseLevel(level)
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     lvl,
		AddSource: lvl == slog.LevelDebug,
	})
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// Logger returns the configured application logger.
func Logger() *slog.Logger {
	if defaultLogger == nil {
		Configure("info")
	}
	return defaultLogger
}

// ParseLevel maps a configuration string to slog.Level. Unknown values default to info.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// IsDebug reports whether the active logger emits debug records.
func IsDebug() bool {
	return Logger().Enabled(nil, slog.LevelDebug)
}
