package config

import "testing"

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
