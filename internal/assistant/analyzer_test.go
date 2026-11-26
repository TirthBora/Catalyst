package assistant_test

import (
	"strings"
	"testing"

	"catalyst/internal/assistant"
)

func TestAnalyzePortConflict(t *testing.T) {
	logs := []string{
		"Starting dev server...",
		"Error: listen EADDRINUSE: address already in use :::3000",
		"Process exited with code 1",
	}

	analysis := assistant.AnalyzeLogs(logs)
	if !analysis.Matched {
		t.Fatal("expected error to be matched")
	}
	if analysis.Category != assistant.CategoryPortConflict {
		t.Errorf("expected CategoryPortConflict, got %s", analysis.Category)
	}
	if !strings.Contains(analysis.Summary, "3000") {
		t.Errorf("expected port 3000 in summary, got: %s", analysis.Summary)
	}
}

func TestAnalyzeNodeMissingModule(t *testing.T) {
	logs := []string{
		"node:internal/modules/cjs/loader:1080",
		"Error: Cannot find module 'lodash'",
		"Require stack:",
		"- D:\\project\\index.js",
	}

	analysis := assistant.AnalyzeLogs(logs)
	if !analysis.Matched || analysis.Category != assistant.CategoryModuleMissing {
		t.Fatalf("expected CategoryModuleMissing, got %+v", analysis)
	}
	if len(analysis.Suggestions) == 0 || !strings.Contains(analysis.Suggestions[0].Command, "lodash") {
		t.Errorf("expected npm install lodash suggestion, got %+v", analysis.Suggestions)
	}
}

func TestAnalyzePythonMissingModule(t *testing.T) {
	logs := []string{
		"Traceback (most recent call last):",
		"  File \"app.py\", line 1, in <module>",
		"ModuleNotFoundError: No module named 'flask'",
	}

	analysis := assistant.AnalyzeLogs(logs)
	if !analysis.Matched || analysis.Category != assistant.CategoryModuleMissing {
		t.Fatalf("expected CategoryModuleMissing, got %+v", analysis)
	}
	if len(analysis.Suggestions) == 0 || !strings.Contains(analysis.Suggestions[0].Command, "flask") {
		t.Errorf("expected pip install flask suggestion, got %+v", analysis.Suggestions)
	}
}

func TestAnalyzeGoUndefined(t *testing.T) {
	logs := []string{
		"# myapp",
		"./server.go:42:15: undefined: NewRouter",
	}

	analysis := assistant.AnalyzeLogs(logs)
	if !analysis.Matched || analysis.Category != assistant.CategorySyntaxError {
		t.Fatalf("expected CategorySyntaxError, got %+v", analysis)
	}
	if analysis.File != "server.go" {
		t.Errorf("expected file server.go, got %s", analysis.File)
	}
}

func TestAnalyzeCleanLogs(t *testing.T) {
	logs := []string{
		"Compiled successfully in 120ms.",
		"Ready at http://localhost:3000",
	}

	analysis := assistant.AnalyzeLogs(logs)
	if analysis.Matched {
		t.Error("expected clean logs to NOT be matched as an error")
	}
}
