package cli_test

import (
	"strings"
	"testing"

	"catalyst/internal/cli"
)

func TestGetInfo(t *testing.T) {
	info := cli.GetInfo()

	if info.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", info.Version)
	}

	if !strings.HasPrefix(info.GoVersion, "go") {
		t.Errorf("expected GoVersion starting with 'go', got %s", info.GoVersion)
	}

	if info.Platform == "" {
		t.Error("expected non-empty platform")
	}
}

func TestFormatVersion(t *testing.T) {
	str := cli.FormatVersion()

	if !strings.Contains(str, "Catalyst v1.0.0") {
		t.Errorf("expected 'Catalyst v1.0.0' in version string, got: %s", str)
	}
}
