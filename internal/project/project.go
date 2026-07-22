package project

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Project represents a Go project detected by Catalyst.
type Project struct {
	Root       string
	Module     string
	EntryPoint string
}

// Detect discovers information about the current Go project.
func Detect() (*Project, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	goModPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goModPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("go.mod not found")
		}
		return nil, fmt.Errorf("check go.mod: %w", err)
	}

	module, err := readModuleName(goModPath)
	if err != nil {
		return nil, err
	}

	entryPoint, err := findEntryPoint(root)
	if err != nil {
		return nil, err
	}

	return &Project{
		Root:       root,
		Module:     module,
		EntryPoint: entryPoint,
	}, nil
}

func readModuleName(goModPath string) (string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", fmt.Errorf("open go.mod: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}

	return "", fmt.Errorf("module declaration not found")
}

func findEntryPoint(root string) (string, error) {
	// First, look inside the cmd directory.
	cmdDir := filepath.Join(root, "cmd")

	if info, err := os.Stat(cmdDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(cmdDir)
		if err != nil {
			return "", fmt.Errorf("read cmd directory: %w", err)
		}

		var candidates []string

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			mainFile := filepath.Join("cmd", entry.Name(), "main.go")

			if _, err := os.Stat(filepath.Join(root, mainFile)); err == nil {
				candidates = append(candidates, mainFile)
			}
		}

		switch len(candidates) {
		case 1:
			return candidates[0], nil

		case 0:
			// fall through and check project root

		default:
			return "", fmt.Errorf(
				"multiple entry points found: %v",
				candidates,
			)
		}
	}

	// Fallback: root/main.go
	if _, err := os.Stat(filepath.Join(root, "main.go")); err == nil {
		return "main.go", nil
	}

	return "", fmt.Errorf("no Go entry point found")
}
