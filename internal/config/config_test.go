package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	content := `
[helper]
path = "/usr/local/bin/iphone-ic-helper"

[devices."a1b2c3d4-uuid"]
name = "Sebs iPhone"
target = "/Users/seb/Pictures/iphone-seb"

[devices."e5f6a7b8-uuid"]
name = "Firmen-iPhone"
target = "/Users/seb/Pictures/iphone-firma"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Helper.Path != "/usr/local/bin/iphone-ic-helper" {
		t.Errorf("helper path = %q, want /usr/local/bin/iphone-ic-helper", cfg.Helper.Path)
	}
	if len(cfg.Devices) != 2 {
		t.Fatalf("devices count = %d, want 2", len(cfg.Devices))
	}
	dev := cfg.Devices["a1b2c3d4-uuid"]
	if dev.Name != "Sebs iPhone" {
		t.Errorf("device name = %q, want 'Sebs iPhone'", dev.Name)
	}
	if dev.Target != "/Users/seb/Pictures/iphone-seb" {
		t.Errorf("device target = %q, want '/Users/seb/Pictures/iphone-seb'", dev.Target)
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.toml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadEmptyConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(cfg.Devices) != 0 {
		t.Errorf("devices count = %d, want 0", len(cfg.Devices))
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")

	cfg := &Config{
		Helper:  HelperConfig{Path: "/usr/local/bin/iphone-ic-helper"},
		Devices: map[string]DeviceConfig{},
	}
	cfg.Devices["new-uuid"] = DeviceConfig{
		Name:   "Test iPhone",
		Target: "/tmp/test-target",
	}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	reloaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load after Save failed: %v", err)
	}
	if reloaded.Devices["new-uuid"].Name != "Test iPhone" {
		t.Errorf("reloaded name = %q, want 'Test iPhone'", reloaded.Devices["new-uuid"].Name)
	}
	if reloaded.Devices["new-uuid"].Target != "/tmp/test-target" {
		t.Errorf("reloaded target = %q, want '/tmp/test-target'", reloaded.Devices["new-uuid"].Target)
	}
}

func TestSavePreservesExistingDevices(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	content := `
[helper]
path = "/usr/local/bin/iphone-ic-helper"

[devices."existing-uuid"]
name = "Existing iPhone"
target = "/tmp/existing"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Devices["new-uuid"] = DeviceConfig{
		Name:   "New iPhone",
		Target: "/tmp/new",
	}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	reloaded, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Devices) != 2 {
		t.Errorf("device count = %d, want 2", len(reloaded.Devices))
	}
	if reloaded.Devices["existing-uuid"].Name != "Existing iPhone" {
		t.Errorf("existing device lost: %q", reloaded.Devices["existing-uuid"].Name)
	}
}
