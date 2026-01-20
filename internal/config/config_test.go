package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_Default(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.JunkPatterns) == 0 {
		t.Error("expected default junk patterns")
	}
	if len(cfg.OrganizeRules) == 0 {
		t.Error("expected default organize rules")
	}
}

func TestLoadConfig_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	yaml := `
junk_patterns:
  - name: "test pattern"
    pattern: "**/test"
    safe: true
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.JunkPatterns) != 1 {
		t.Errorf("expected 1 pattern, got %d", len(cfg.JunkPatterns))
	}
	if cfg.JunkPatterns[0].Name != "test pattern" {
		t.Errorf("expected 'test pattern', got %s", cfg.JunkPatterns[0].Name)
	}
}

func TestDefaultConfig_HasJunkPatterns(t *testing.T) {
	cfg := DefaultConfig()
	if len(cfg.JunkPatterns) == 0 {
		t.Error("expected default junk patterns")
	}
	// Check for node_modules pattern
	found := false
	for _, p := range cfg.JunkPatterns {
		if p.Name == "node_modules" {
			found = true
			if p.Pattern != "**/node_modules" {
				t.Errorf("node_modules pattern = %s, want **/node_modules", p.Pattern)
			}
			if !p.Safe {
				t.Error("node_modules should be marked safe")
			}
			break
		}
	}
	if !found {
		t.Error("expected node_modules pattern in defaults")
	}
}

func TestDefaultConfig_HasOrganizeRules(t *testing.T) {
	cfg := DefaultConfig()
	if len(cfg.OrganizeRules) == 0 {
		t.Error("expected default organize rules")
	}
}

func TestDefaultConfig_HasDeletionSettings(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Deletion.TrashThreshold == "" {
		t.Error("expected trash threshold")
	}
	if len(cfg.Deletion.AlwaysTrash) == 0 {
		t.Error("expected always_trash list")
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load() error = %v, expected defaults for non-existent file", err)
	}
	if len(cfg.JunkPatterns) == 0 {
		t.Error("expected default junk patterns for non-existent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")

	invalidYAML := `
junk_patterns:
  - name: "test"
    pattern: [invalid
`
	if err := os.WriteFile(cfgPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	if path == "" {
		t.Error("DefaultPath() returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("DefaultPath() = %s, expected absolute path", path)
	}
}

func TestDataPath(t *testing.T) {
	path := DataPath()
	if path == "" {
		t.Error("DataPath() returned empty string")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("DataPath() = %s, expected absolute path", path)
	}
}
