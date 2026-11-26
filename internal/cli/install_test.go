package cli_test

import (
	"strings"
	"testing"

	"catalyst/internal/cli"
)

func TestInstallCLI(t *testing.T) {
	target, err := cli.InstallCLI()
	if err != nil {
		t.Fatalf("InstallCLI returned unexpected error: %v", err)
	}

	if !strings.Contains(target, "catalyst") {
		t.Errorf("expected target path to contain 'catalyst', got %s", target)
	}
}
