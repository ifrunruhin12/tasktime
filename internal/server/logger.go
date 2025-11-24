package server

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	level      LogLevel
	output     io.Writer
	mu         sync.Mutex
	file       *os.File
	maxSize    int64 // Maximum file size in bytes before rotation
	maxBackups int   // Maximum number of old log files to keep
}

func NewLogger(level LogLevel, logFile string) (*Logger, error) {
	logger := &Logger{
		level:      level,
		maxSize:    100 * 1024 * 1024, // 100MB default
		maxBackups: 10,
	}

	if logFile == "" {
		logger.output = os.Stdout
		return logger, nil
	}

	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger.file = file
	logger.output = file

	return logger, nil
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) log(level LogLevel, message string, context map[string]interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.rotateIfNeeded()
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s | %s", level.String(), timestamp, message)

	if len(context) > 0 {
		logLine += " |"
		for key, value := range context {
			logLine += fmt.Sprintf(" %s=%v", key, value)
		}
	}

	logLine += "\n"

	l.output.Write([]byte(logLine))
}

func (l *Logger) rotateIfNeeded() {
	if l.file == nil {
		return
	}

	info, err := l.file.Stat()
	if err != nil {
		return
	}

	if info.Size() < l.maxSize {
		return
	}

	l.file.Close()

	logPath := l.file.Name()
	for i := l.maxBackups - 1; i >= 0; i-- {
		oldPath := fmt.Sprintf("%s.%d", logPath, i)
		newPath := fmt.Sprintf("%s.%d", logPath, i+1)

		if i == l.maxBackups-1 {
			os.Remove(newPath)
		}

		if _, err := os.Stat(oldPath); err == nil {
			os.Rename(oldPath, newPath)
		}
	}

	os.Rename(logPath, fmt.Sprintf("%s.0", logPath))

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		l.output = os.Stdout
		return
	}

	l.file = file
	l.output = file
}

func (l *Logger) Debug(message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.log(DEBUG, message, ctx)
}

func (l *Logger) Info(message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.log(INFO, message, ctx)
}

func (l *Logger) Warn(message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.log(WARN, message, ctx)
}

func (l *Logger) Error(message string, context ...map[string]interface{}) {
	ctx := make(map[string]interface{})
	if len(context) > 0 {
		ctx = context[0]
	}
	l.log(ERROR, message, ctx)
}

func ParseLogLevel(level string) LogLevel {
	switch level {
	case "DEBUG", "debug":
		return DEBUG
	case "INFO", "info":
		return INFO
	case "WARN", "warn", "WARNING", "warning":
		return WARN
	case "ERROR", "error":
		return ERROR
	default:
		return INFO
	}
}
