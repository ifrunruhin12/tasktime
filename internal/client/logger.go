package client

import (
	"log/slog"
	"os"
	"path/filepath"
)

var logger *slog.Logger

// InitLogger initializes the global slog logger for the client
func InitLogger(debugMode bool) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Create .tasktime directory if it doesn't exist
	logDir := filepath.Join(homeDir, ".tasktime")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Open log file
	logPath := filepath.Join(logDir, "client.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	// Set log level based on debug mode
	level := slog.LevelInfo
	if debugMode {
		level = slog.LevelDebug
	}

	// Create structured logger with custom options
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Customize the time format
			if a.Key == slog.TimeKey {
				return slog.Attr{
					Key:   a.Key,
					Value: slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05")),
				}
			}
			return a
		},
	}

	handler := slog.NewTextHandler(file, opts)
	logger = slog.New(handler).With("component", "client")

	return nil
}

// CloseLogger closes any resources (for compatibility, slog handles cleanup automatically)
func CloseLogger() error {
	// slog handles cleanup automatically, but we could close the file if we kept a reference
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
func LogHTTPRequest(method, url string, statusCode int, duration int64) {
	GetLogger().Debug("HTTP request",
		"method", method,
		"url", url,
		"status", statusCode,
		"duration_ms", duration,
	)
}

func LogWebSocketEvent(event string, args ...any) {
	allArgs := append([]any{"event", event}, args...)
	GetLogger().Info("WebSocket event", allArgs...)
}

func LogAuthEvent(event string, args ...any) {
	allArgs := append([]any{"event", event}, args...)
	GetLogger().Info("Auth event", allArgs...)
}

func LogConfigEvent(event string, args ...any) {
	allArgs := append([]any{"event", event}, args...)
	GetLogger().Info("Config event", allArgs...)
}