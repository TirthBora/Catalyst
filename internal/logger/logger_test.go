package logger_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"catalyst/internal/logger"
)

func TestLoggerLevels(t *testing.T) {
	tempDir := t.TempDir()

	l := logger.New(logger.LevelInfo, tempDir)
	defer l.Close()

	l.Debug("this is debug")
	l.Info("this is info")
	l.Warn("this is warn")
	l.Error("this is error")

	l.Close()

	// Read persisted log file
	data, err := os.ReadFile(filepath.Join(tempDir, "catalyst.log"))
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	content := string(data)

	if strings.Contains(content, "this is debug") {
		t.Error("expected debug message to be filtered out at Info level")
	}

	if !strings.Contains(content, "this is info") {
		t.Error("expected info message to be written")
	}

	if !strings.Contains(content, "this is warn") {
		t.Error("expected warn message to be written")
	}

	if !strings.Contains(content, "this is error") {
		t.Error("expected error message to be written")
	}
}

func TestLoggerOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	l := logger.NewWithWriter(logger.LevelDebug, buf, "")
	defer l.Close()

	l.Info("direct test %d", 42)
	l.Close()

	if !strings.Contains(buf.String(), "direct test 42") {
		t.Errorf("expected buffer to receive logged message, got: %s", buf.String())
	}
}
