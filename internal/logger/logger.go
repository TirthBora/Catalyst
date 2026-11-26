package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level defines the severity of a log message.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// Logger provides thread-safe leveled and persistent file logging.
type Logger struct {
	minLevel Level
	out      io.Writer
	file     *os.File
	mu       sync.Mutex
}

// New creates a new Logger writing to stdout and optionally to a log file.
func New(minLevel Level, logDir string) *Logger {
	return NewWithWriter(minLevel, os.Stdout, logDir)
}

// NewWithWriter creates a new Logger writing to custom out and optionally to a log file.
func NewWithWriter(minLevel Level, out io.Writer, logDir string) *Logger {
	l := &Logger{
		minLevel: minLevel,
		out:      out,
	}

	if logDir != "" {
		_ = os.MkdirAll(logDir, 0755)
		logPath := filepath.Join(logDir, "catalyst.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			l.file = f
		}
	}

	return l
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.minLevel {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	ts := time.Now().UTC().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	entry := fmt.Sprintf("[%s] [%-5s] %s\n", ts, level.String(), msg)

	if l.out != nil {
		_, _ = l.out.Write([]byte(entry))
	}
	if l.file != nil {
		_, _ = l.file.Write([]byte(entry))
	}
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LevelDebug, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LevelInfo, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LevelWarn, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LevelError, format, args...)
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}
