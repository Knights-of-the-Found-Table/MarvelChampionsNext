// Package logx configures the process-wide structured logger from
// MC_LOG_LEVEL and provides a Fatal helper (log/slog has none).
package logx

import (
	"log/slog"
	"os"
	"strings"
)

// Init sets the default slog logger (text handler on stderr) from
// MC_LOG_LEVEL: debug | info | warn | error. Unset or invalid values fall
// back to info; an invalid value is reported through the configured logger.
// Call after loading .env so the variable may come from either source.
func Init() {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("MC_LOG_LEVEL")))
	var level slog.Level
	valid := true
	switch raw {
	case "debug":
		level = slog.LevelDebug
	case "", "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
		valid = false
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
	if !valid {
		slog.Warn("invalid MC_LOG_LEVEL; using info", "value", raw)
	}
}

// Fatal logs at error level and exits.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
