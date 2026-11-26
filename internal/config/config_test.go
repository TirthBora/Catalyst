package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"catalyst/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.Port != "9999" {
		t.Errorf("expected default port 9999, got %s", cfg.Port)
	}
	if cfg.TargetURL != "/preview/" {
		t.Errorf("expected default target /preview/, got %s", cfg.TargetURL)
	}
	if !cfg.AutoReload {
		t.Error("expected default AutoReload to be true")
	}
}

func TestConfigPersistence(t *testing.T) {
	dir := t.TempDir()

	mgr := config.NewManager(dir)
	initial := mgr.Get()
	if initial.Port != "9999" {
		t.Fatalf("expected initial port 9999, got %s", initial.Port)
	}

	// Update configuration
	updated := config.Config{
		Port:          "8888",
		TargetURL:     "/proxy/5173/",
		CustomIgnore:  []string{"cache", "logs"},
		DefaultDevice: "tablet",
		AutoReload:    false,
		Theme:         "light",
	}

	if err := mgr.Update(updated); err != nil {
		t.Fatalf("failed to update config: %v", err)
	}

	// Create new manager instance for the same directory to verify disk reload
	mgr2 := config.NewManager(dir)
	loaded := mgr2.Get()

	if loaded.Port != "8888" {
		t.Errorf("expected port 8888, got %s", loaded.Port)
	}
	if loaded.TargetURL != "/proxy/5173/" {
		t.Errorf("expected target /proxy/5173/, got %s", loaded.TargetURL)
	}
	if loaded.DefaultDevice != "tablet" {
		t.Errorf("expected device tablet, got %s", loaded.DefaultDevice)
	}
	if loaded.AutoReload != false {
		t.Error("expected AutoReload to be false")
	}
	if len(loaded.CustomIgnore) != 2 || loaded.CustomIgnore[0] != "cache" {
		t.Errorf("expected CustomIgnore [cache logs], got %+v", loaded.CustomIgnore)
	}
}

func TestRootConfigFilePriority(t *testing.T) {
	dir := t.TempDir()

	// Place catalyst.config.json directly in root
	content := `{
		"port": "7777",
		"targetUrl": "/custom/"
	}`
	if err := os.WriteFile(filepath.Join(dir, "catalyst.config.json"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := config.NewManager(dir)
	cfg := mgr.Get()

	if cfg.Port != "7777" {
		t.Errorf("expected port 7777 from root config, got %s", cfg.Port)
	}
	if cfg.TargetURL != "/custom/" {
		t.Errorf("expected target /custom/ from root config, got %s", cfg.TargetURL)
	}
}
