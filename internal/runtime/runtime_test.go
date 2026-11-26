package runtime_test

import (
	"os"
	"path/filepath"
	"testing"

	"catalyst/internal/runtime"
)

func TestDiagnose(t *testing.T) {
	dir := t.TempDir()

	// 1. Diagnose empty directory
	report := runtime.Diagnose(dir)

	if len(report.Toolchains) == 0 {
		t.Fatal("expected at least one probed toolchain")
	}

	// Go must be installed in the test environment
	var foundGo bool
	for _, tc := range report.Toolchains {
		if tc.Name == "Go" {
			foundGo = true
			if !tc.Installed {
				t.Error("expected Go to be marked as installed")
			}
			if tc.Version == "" {
				t.Error("expected non-empty Go version")
			}
		}
	}
	if !foundGo {
		t.Error("Go toolchain probe missing from report")
	}

	// 2. Test Node.js dependency warning when node_modules is missing
	pkgFile := filepath.Join(dir, "package.json")
	if err := os.WriteFile(pkgFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	report2 := runtime.Diagnose(dir)
	var foundNodeDepWarning bool
	for _, c := range report2.Checks {
		if c.Name == "Node Dependencies" && c.Status == "warning" {
			foundNodeDepWarning = true
			break
		}
	}
	if !foundNodeDepWarning {
		t.Error("expected Node Dependencies warning for missing node_modules")
	}

	// 3. Test Node.js dependency OK when node_modules is present
	nodeModules := filepath.Join(dir, "node_modules")
	if err := os.Mkdir(nodeModules, 0755); err != nil {
		t.Fatal(err)
	}

	report3 := runtime.Diagnose(dir)
	var foundNodeDepOK bool
	for _, c := range report3.Checks {
		if c.Name == "Node Dependencies" && c.Status == "ok" {
			foundNodeDepOK = true
			break
		}
	}
	if !foundNodeDepOK {
		t.Error("expected Node Dependencies ok when node_modules exists")
	}
}
