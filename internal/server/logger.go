package server

import (
	"log/slog"
	"os"
	"path/filepath"
)

var logger *slog.Logger

// InitLogger initializes the global slog logger for the server
func InitLogger(level string, logFile string) error {
	// Parse log level
	var slogLevel slog.Level
	switch level {
	case "DEBUG", "debug":
		slogLevel = slog.LevelDebug
	case "INFO", "info":
		slogLevel = slog.LevelInfo
	case "WARN", "warn", "WARNING", "warning":
		slogLevel = slog.LevelWarn
	case "ERROR", "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	var handler slog.Handler

	if logFile == "" {
		// Log to stdout
		opts := &slog.HandlerOptions{
			Level: slogLevel,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Attr{
						Key:   a.Key,
						Value: slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05")),
					}
				}
				return a
			},
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// Log to file
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return err
		}

		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		opts := &slog.HandlerOptions{
			Level: slogLevel,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Attr{
						Key:   a.Key,
						Value: slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05")),
					}
				}
				return a
			},
		}
		handler = slog.NewTextHandler(file, opts)
	}

	logger = slog.New(handler).With("component", "server")
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *slog.Logger {
	if logger == nil {
		// Fallback to default logger if not initialized
		return slog.Default()
	}
	return logger
}

// Convenience functions for logging
func LogDebug(msg string, args ...any) {
	GetLogger().Debug(msg, args...)
}

func LogInfo(msg string, args ...any) {
	GetLogger().Info(msg, args...)
}

func LogWarn(msg string, args ...any) {
	GetLogger().Warn(msg, args...)
}

func LogError(msg string, args ...any) {
	GetLogger().Error(msg, args...)
}

// Specialized logging functions
func LogHTTPRequest(method, path string, statusCode int, duration int64, args ...any) {
	allArgs := append([]any{
		"method", method,
		"path", path,
		"status", statusCode,
		"duration_ms", duration,
	}, args...)
	GetLogger().Info("HTTP request", allArgs...)
}

func LogAuthEvent(event string, args ...any) {
	allArgs := append([]any{"event", event}, args...)
	GetLogger().Info("Auth event", allArgs...)
}

func LogWebSocketEvent(event string, args ...any) {
	allArgs := append([]any{"event", event}, args...)
	GetLogger().Info("WebSocket event", allArgs...)
}

func LogDatabaseEvent(event string, args ...any) {
	allArgs := append([]any{"event", event}, args...)
	GetLogger().Info("Database event", allArgs...)
}
