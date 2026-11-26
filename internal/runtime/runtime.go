package runtime

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Toolchain contains information about an installed compiler or runtime.
type Toolchain struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Path      string `json:"path,omitempty"`
}

// CheckItem represents a specific diagnostic verification.
type CheckItem struct {
	Category string `json:"category"` // "toolchain", "workspace", "network"
	Name     string `json:"name"`
	Status   string `json:"status"`   // "ok", "warning", "error"
	Message  string `json:"message"`
}

// DoctorReport contains full environment diagnostics.
type DoctorReport struct {
	Toolchains []Toolchain `json:"toolchains"`
	Checks     []CheckItem `json:"checks"`
	Timestamp  time.Time   `json:"timestamp"`
}

// checkToolchain probes the PATH for an executable and extracts its version.
func checkToolchain(name, cmdName string, versionArgs ...string) Toolchain {
	path, err := exec.LookPath(cmdName)
	if err != nil {
		return Toolchain{Name: name, Installed: false}
	}

	tc := Toolchain{
		Name:      name,
		Installed: true,
		Path:      path,
	}

	if len(versionArgs) > 0 {
		cmd := exec.Command(cmdName, versionArgs...)
		if out, err := cmd.Output(); err == nil {
			firstLine := strings.Split(strings.TrimSpace(string(out)), "\n")[0]
			tc.Version = strings.TrimSpace(firstLine)
		}
	}

	return tc
}

// Diagnose inspects the host toolchains and workspace configuration.
func Diagnose(rootDir string) DoctorReport {
	toolchains := []Toolchain{
		checkToolchain("Go", "go", "version"),
		checkToolchain("Node.js", "node", "--version"),
		checkToolchain("npm", "npm", "--version"),
		checkToolchain("Git", "git", "--version"),
		checkToolchain("Python", "python", "--version"),
		checkToolchain("Rust", "rustc", "--version"),
	}

	var checks []CheckItem

	// 1. Toolchain health checks
	for _, tc := range toolchains {
		if tc.Installed {
			checks = append(checks, CheckItem{
				Category: "toolchain",
				Name:     tc.Name,
				Status:   "ok",
				Message:  tc.Version,
			})
		} else {
			checks = append(checks, CheckItem{
				Category: "toolchain",
				Name:     tc.Name,
				Status:   "warning",
				Message:  "Not detected in PATH",
			})
		}
	}

	// 2. Workspace checks
	if rootDir != "" {
		// Git repository check
		if _, err := os.Stat(filepath.Join(rootDir, ".git")); err == nil {
			checks = append(checks, CheckItem{
				Category: "workspace",
				Name:     "Git Repository",
				Status:   "ok",
				Message:  "Initialized (.git present)",
			})
		} else {
			checks = append(checks, CheckItem{
				Category: "workspace",
				Name:     "Git Repository",
				Status:   "warning",
				Message:  "Not a git repository",
			})
		}

		// Node.js dependencies check
		if _, err := os.Stat(filepath.Join(rootDir, "package.json")); err == nil {
			if _, err := os.Stat(filepath.Join(rootDir, "node_modules")); err == nil {
				checks = append(checks, CheckItem{
					Category: "workspace",
					Name:     "Node Dependencies",
					Status:   "ok",
					Message:  "node_modules present",
				})
			} else {
				checks = append(checks, CheckItem{
					Category: "workspace",
					Name:     "Node Dependencies",
					Status:   "warning",
					Message:  "package.json found but node_modules is missing (run npm install)",
				})
			}
		}

		// Python venv check
		if _, err := os.Stat(filepath.Join(rootDir, "requirements.txt")); err == nil {
			hasVenv := false
			for _, v := range []string{".venv", "venv", "env"} {
				if _, err := os.Stat(filepath.Join(rootDir, v)); err == nil {
					hasVenv = true
					break
				}
			}
			if hasVenv {
				checks = append(checks, CheckItem{
					Category: "workspace",
					Name:     "Python Environment",
					Status:   "ok",
					Message:  "Virtual environment directory detected",
				})
			} else {
				checks = append(checks, CheckItem{
					Category: "workspace",
					Name:     "Python Environment",
					Status:   "warning",
					Message:  "requirements.txt found but no .venv directory detected",
				})
			}
		}
	}

	return DoctorReport{
		Toolchains: toolchains,
		Checks:     checks,
		Timestamp:  time.Now().UTC(),
	}
}
