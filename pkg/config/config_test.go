package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	// NerdFontsVersion can be empty (disabled), "2", or "3" - all valid

	// Theme colors should be set (either from defaults or config file)
	if len(cfg.UI.Theme.ActiveBorderColor) == 0 {
		t.Error("ActiveBorderColor should have a value")
	}

	if len(cfg.UI.Theme.InactiveBorderColor) == 0 {
		t.Error("InactiveBorderColor should have a value")
	}
}

func TestConfigStructure(t *testing.T) {
	cfg := &Config{
		UI: UIConfig{
			NerdFontsVersion: "2",
			Theme: ThemeConfig{
				ActiveBorderColor:   []string{"red", "bold"},
				InactiveBorderColor: []string{"gray"},
				OptionsTextColor:    []string{"white"},
				SelectedLineBgColor: []string{"#ff0000"},
			},
		},
	}

	if cfg.UI.NerdFontsVersion != "2" {
		t.Error("NerdFontsVersion not set correctly")
	}

	if len(cfg.UI.Theme.ActiveBorderColor) != 2 {
		t.Error("ActiveBorderColor should support multiple values")
	}

	if cfg.UI.Theme.SelectedLineBgColor[0] != "#ff0000" {
		t.Error("Should support hex color values")
	}
}

// isolate points every config location at empty temp dirs and returns them
func isolate(t *testing.T) (home, xdg, cwd string) {
	t.Helper()
	home, xdg, cwd = t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("LAZYFIRE_CONFIG_FILE", "")
	t.Chdir(cwd)
	return home, xdg, cwd
}

func writeConfig(t *testing.T, path, activeColor string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "ui:\n  theme:\n    activeBorderColor:\n      - \"" + activeColor + "\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func activeColor(t *testing.T) string {
	t.Helper()
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	return cfg.UI.Theme.ActiveBorderColor[0]
}

func TestLoadConfigDefaultsWithoutFile(t *testing.T) {
	isolate(t)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UI.NerdFontsVersion != "3" || !cfg.UI.ShowIcons || cfg.UI.Theme.ActiveBorderColor[0] != "#ed8796" {
		t.Errorf("unexpected defaults: %+v", cfg.UI)
	}
}

func TestLoadConfigSearchOrder(t *testing.T) {
	home, xdg, cwd := isolate(t)

	writeConfig(t, filepath.Join(cwd, "config.yaml"), "cwd")
	if got := activeColor(t); got != "cwd" {
		t.Errorf("./config.yaml: got %q", got)
	}

	writeConfig(t, filepath.Join(home, ".lazyfire", "config.yaml"), "legacy")
	if got := activeColor(t); got != "legacy" {
		t.Errorf("~/.lazyfire should win over ./config.yaml, got %q", got)
	}

	writeConfig(t, filepath.Join(xdg, "lazyfire", "config.yml"), "xdg")
	if got := activeColor(t); got != "xdg" {
		t.Errorf("$XDG_CONFIG_HOME/lazyfire should win over ~/.lazyfire, got %q", got)
	}

	explicit := filepath.Join(t.TempDir(), "custom.yml")
	writeConfig(t, explicit, "explicit")
	t.Setenv("LAZYFIRE_CONFIG_FILE", explicit)
	if got := activeColor(t); got != "explicit" {
		t.Errorf("LAZYFIRE_CONFIG_FILE should win, got %q", got)
	}
}

func TestLoadConfigDefaultsToDotConfig(t *testing.T) {
	home, _, _ := isolate(t)
	t.Setenv("XDG_CONFIG_HOME", "")
	writeConfig(t, filepath.Join(home, ".config", "lazyfire", "config.yml"), "dotconfig")
	if got := activeColor(t); got != "dotconfig" {
		t.Errorf("~/.config/lazyfire/config.yml: got %q", got)
	}
}

func TestLoadConfigPartialFileKeepsDefaults(t *testing.T) {
	_, xdg, _ := isolate(t)
	writeConfig(t, filepath.Join(xdg, "lazyfire", "config.yml"), "#ffffff")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UI.NerdFontsVersion != "3" || cfg.UI.Theme.InactiveBorderColor[0] != "#5f626b" {
		t.Errorf("settings missing from the file should keep defaults: %+v", cfg.UI)
	}
}

func TestLoadConfigMissingExplicitFile(t *testing.T) {
	isolate(t)
	t.Setenv("LAZYFIRE_CONFIG_FILE", filepath.Join(t.TempDir(), "missing.yml"))
	if _, err := LoadConfig(); err == nil {
		t.Error("a missing LAZYFIRE_CONFIG_FILE should be an error")
	}
}

// The example file documents the defaults, so it must not drift from them
func TestExampleConfigMatchesDefaults(t *testing.T) {
	example, err := filepath.Abs("../../config.example.yaml")
	if err != nil {
		t.Fatal(err)
	}
	isolate(t)
	defaults, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("LAZYFIRE_CONFIG_FILE", example)
	fromExample, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(defaults, fromExample) {
		t.Errorf("config.example.yaml differs from the defaults:\nexample:  %+v\ndefaults: %+v", fromExample.UI, defaults.UI)
	}
}
