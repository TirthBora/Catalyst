package analyzer

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// LineDiff details the specific lines added and removed during a file change.
type LineDiff struct {
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Summary string   `json:"summary"` // e.g. "+3, -1 lines"
}

// DiffTracker maintains an in-memory cache of file contents to compute delta changes.
type DiffTracker struct {
	cache map[string]string
	mu    sync.RWMutex
}

func NewDiffTracker() *DiffTracker {
	return &DiffTracker{
		cache: make(map[string]string),
	}
}

// ComputeDiff calculates the line additions and deletions for a modified file.
func (dt *DiffTracker) ComputeDiff(fullPath string) LineDiff {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return LineDiff{}
	}
	newContent := string(data)
	oldContent, exists := dt.cache[fullPath]
	dt.cache[fullPath] = newContent

	if !exists {
		return LineDiff{
			Summary: "File modified",
		}
	}

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	oldSet := make(map[string]int)
	for _, l := range oldLines {
		t := strings.TrimSpace(l)
		if t != "" {
			oldSet[t]++
		}
	}

	newSet := make(map[string]int)
	for _, l := range newLines {
		t := strings.TrimSpace(l)
		if t != "" {
			newSet[t]++
		}
	}

	var added []string
	var removed []string

	for _, l := range newLines {
		t := strings.TrimSpace(l)
		if t != "" && oldSet[t] == 0 && len(added) < 10 {
			added = append(added, strings.TrimRight(l, "\r"))
		}
	}

	for _, l := range oldLines {
		t := strings.TrimSpace(l)
		if t != "" && newSet[t] == 0 && len(removed) < 10 {
			removed = append(removed, strings.TrimRight(l, "\r"))
		}
	}

	summary := fmt.Sprintf("+%d, -%d lines", len(added), len(removed))
	return LineDiff{
		Added:   added,
		Removed: removed,
		Summary: summary,
	}
}
