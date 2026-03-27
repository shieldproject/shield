package log

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// SetupLogging initializes the default slog logger at the given level.
// Accepted level strings: "error", "warn", "warning", "info", "debug".
func SetupLogging(level string) {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn", "warning":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: l})
	slog.SetDefault(slog.New(h))
}

func Infof(f string, args ...interface{}) {
	slog.Default().Info(fmt.Sprintf(f, args...))
}

func Debugf(f string, args ...interface{}) {
	slog.Default().Debug(fmt.Sprintf(f, args...))
}

func Errorf(f string, args ...interface{}) {
	slog.Default().Error(fmt.Sprintf(f, args...))
}

func Warnf(f string, args ...interface{}) {
	slog.Default().Warn(fmt.Sprintf(f, args...))
}

// Warningf is an alias for Warnf to match go-log's API.
func Warningf(f string, args ...interface{}) {
	Warnf(f, args...)
}

// Alertf maps go-log's alert level to warn (slog has no alert level).
func Alertf(f string, args ...interface{}) {
	Warnf(f, args...)
}

// IsDebug reports whether the default logger is at debug level.
func IsDebug() bool {
	return slog.Default().Enabled(nil, slog.LevelDebug)
}
