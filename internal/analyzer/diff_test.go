package analyzer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"catalyst/internal/analyzer"
)

func TestComputeDiff(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")

	// 1. Initial write
	_ = os.WriteFile(file, []byte("line 1\nline 2\nline 3\n"), 0644)

	tracker := analyzer.NewDiffTracker()
	diff1 := tracker.ComputeDiff(file)
	if diff1.Summary != "File modified" {
		t.Errorf("expected initial summary 'File modified', got %s", diff1.Summary)
	}

	// 2. Modify file (add line 4, change line 2)
	_ = os.WriteFile(file, []byte("line 1\nline 2 MODIFIED\nline 3\nline 4 NEW\n"), 0644)

	diff2 := tracker.ComputeDiff(file)
	if len(diff2.Added) == 0 {
		t.Error("expected added lines in diff")
	}

	if len(diff2.Removed) == 0 {
		t.Error("expected removed lines in diff")
	}

	if !strings.Contains(diff2.Summary, "+") || !strings.Contains(diff2.Summary, "-") {
		t.Errorf("expected summary with + and -, got %s", diff2.Summary)
	}
}
