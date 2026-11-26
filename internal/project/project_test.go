package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"catalyst/internal/project"
)

func TestDetectGo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module myapp\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info := project.Detect(dir)
	if info.Type != project.TypeGo {
		t.Errorf("expected TypeGo, got %s", info.Type)
	}
	if info.DefaultPort != "8080" {
		t.Errorf("expected port 8080, got %s", info.DefaultPort)
	}
	if len(info.Scripts) == 0 || info.Scripts[0].Command != "go" {
		t.Errorf("expected go scripts to be extracted, got %+v", info.Scripts)
	}
}

func TestDetectVite(t *testing.T) {
	dir := t.TempDir()
	pkgContent := `{
		"name": "vite-dashboard",
		"devDependencies": {
			"vite": "^5.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgContent), 0644); err != nil {
		t.Fatal(err)
	}

	info := project.Detect(dir)
	if info.Type != project.TypeVite {
		t.Errorf("expected TypeVite, got %s", info.Type)
	}
	if info.Name != "vite-dashboard" {
		t.Errorf("expected name vite-dashboard, got %s", info.Name)
	}
	if info.DefaultPort != "5173" {
		t.Errorf("expected port 5173, got %s", info.DefaultPort)
	}
}

func TestDetectNext(t *testing.T) {
	dir := t.TempDir()
	pkgContent := `{
		"name": "my-next-app",
		"scripts": {
			"dev": "next dev",
			"build": "next build"
		},
		"dependencies": {
			"next": "^14.0.0",
			"react": "^18.0.0"
		}
	}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgContent), 0644); err != nil {
		t.Fatal(err)
	}

	info := project.Detect(dir)
	if info.Type != project.TypeNext {
		t.Errorf("expected TypeNext, got %s", info.Type)
	}
	if info.DefaultPort != "3000" {
		t.Errorf("expected port 3000, got %s", info.DefaultPort)
	}
	if len(info.Scripts) != 2 {
		t.Errorf("expected 2 extracted scripts from package.json, got %d", len(info.Scripts))
	}
}

func TestDetectPythonFlask(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask==3.0.0\ngunicorn\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info := project.Detect(dir)
	if info.Type != project.TypePython {
		t.Errorf("expected TypePython, got %s", info.Type)
	}
	if info.Framework != "Flask" {
		t.Errorf("expected Flask, got %s", info.Framework)
	}
	if info.DefaultPort != "5000" {
		t.Errorf("expected port 5000, got %s", info.DefaultPort)
	}
}

func TestDetectRust(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"cli\"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info := project.Detect(dir)
	if info.Type != project.TypeRust {
		t.Errorf("expected TypeRust, got %s", info.Type)
	}
}

func TestDetectStatic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!DOCTYPE html>"), 0644); err != nil {
		t.Fatal(err)
	}

	info := project.Detect(dir)
	if info.Type != project.TypeStatic {
		t.Errorf("expected TypeStatic, got %s", info.Type)
	}
}
