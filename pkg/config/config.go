// Package config handles loading and parsing of LazyFire configuration.
// See LoadConfig for where the configuration file is looked up.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config is the root configuration structure for LazyFire.
type Config struct {
	UI       UIConfig       `mapstructure:"ui"`
	Emulator EmulatorConfig `mapstructure:"emulator"`
}

// EmulatorConfig contains settings for connecting to local Firebase emulators.
type EmulatorConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	ProjectID     string `mapstructure:"projectId"`     // Required when emulator is enabled
	FirestoreHost string `mapstructure:"firestoreHost"` // Default: localhost:8080
}

// UIConfig contains user interface configuration options.
type UIConfig struct {
	ShowIcons        bool        `mapstructure:"showIcons"`        // Enable/disable icons
	NerdFontsVersion string      `mapstructure:"nerdFontsVersion"` // "2", "3", or "" to disable
	Theme            ThemeConfig `mapstructure:"theme"`
}

// ThemeConfig defines the color scheme for the terminal UI.
// Colors can be specified as:
//   - Named colors: "cyan", "blue", "red", "green", "yellow", "magenta", "white", "black", "default"
//   - Hex colors: "#ed8796"
//   - 256-color numbers: "0" to "255"
//   - Attributes: "bold", "underline", "reverse"
type ThemeConfig struct {
	// ActiveBorderColor is the color of the focused panel's border and title
	ActiveBorderColor []string `mapstructure:"activeBorderColor"`
	// InactiveBorderColor is the color of unfocused panel borders
	InactiveBorderColor []string `mapstructure:"inactiveBorderColor"`
	// OptionsTextColor is the color of help text in the footer
	OptionsTextColor []string `mapstructure:"optionsTextColor"`
	// SelectedLineBgColor is the background color of the highlighted row
	SelectedLineBgColor []string `mapstructure:"selectedLineBgColor"`
}

// LoadConfig returns the built-in defaults, overridden by the first config
// file found, like lazygit:
//  1. the file named by the LAZYFIRE_CONFIG_FILE environment variable
//  2. $XDG_CONFIG_HOME/lazyfire/config.yml (~/.config/lazyfire/ by default)
//  3. ~/.lazyfire/config.yaml
//  4. ./config.yaml
//
// Settings missing from the file keep their defaults. Both .yml and .yaml work.
func LoadConfig() (*Config, error) {
	// Default configuration
	config := &Config{
		UI: UIConfig{
			ShowIcons:        true,
			NerdFontsVersion: "3", // Default to Nerd Fonts v3, set to "" to disable
			Theme: ThemeConfig{
				ActiveBorderColor:   []string{"#ed8796", "bold"},
				InactiveBorderColor: []string{"#5f626b"},
				OptionsTextColor:    []string{"#8aadf4"},
				SelectedLineBgColor: []string{"#494d64", "bold"},
			},
		},
	}

	v := viper.New()
	v.SetConfigType("yaml")

	// A file named explicitly must exist and parse
	if path := os.Getenv("LAZYFIRE_CONFIG_FILE"); path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return config, fmt.Errorf("LAZYFIRE_CONFIG_FILE: %w", err)
		}
		return config, v.Unmarshal(config)
	}

	v.SetConfigName("config")
	for _, dir := range configDirs() {
		v.AddConfigPath(dir)
	}

	// Read and parse config file if it exists
	if err := v.ReadInConfig(); err == nil {
		if err := v.Unmarshal(config); err != nil {
			return config, err
		}
	}

	return config, nil
}

// configDirs lists the directories searched for config.yml, in order
func configDirs() []string {
	var dirs []string
	home, homeErr := os.UserHomeDir()
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" && homeErr == nil {
		configHome = filepath.Join(home, ".config")
	}
	if configHome != "" {
		dirs = append(dirs, filepath.Join(configHome, "lazyfire"))
	}
	if homeErr == nil {
		dirs = append(dirs, filepath.Join(home, ".lazyfire"))
	}
	return append(dirs, ".")
}
