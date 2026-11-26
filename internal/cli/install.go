package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// InstallCLI copies the running catalyst executable to the user's bin directory on PATH.
func InstallCLI() (string, error) {
	currentExe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to determine executable path: %w", err)
	}

	targetDir := ""
	home, _ := os.UserHomeDir()

	if runtime.GOOS == "windows" {
		targetDir = filepath.Join(home, "bin")
	} else {
		targetDir = filepath.Join(home, ".local", "bin")
		if _, err := os.Stat(targetDir); err != nil {
			targetDir = filepath.Join(home, "bin")
		}
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
	}

	targetExt := ""
	if runtime.GOOS == "windows" {
		targetExt = ".exe"
	}
	targetPath := filepath.Join(targetDir, "catalyst"+targetExt)

	// Avoid copying onto itself
	if filepath.Clean(currentExe) == filepath.Clean(targetPath) {
		return targetPath, nil
	}

	src, err := os.Open(currentExe)
	if err != nil {
		return "", fmt.Errorf("failed to read source binary: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to write destination binary: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy binary: %w", err)
	}

	return targetPath, nil
}
