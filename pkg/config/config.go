// Package config handles loading and parsing of LazyFire configuration.
// Configuration is loaded from ~/.lazyfire/config.yaml or ./config.yaml
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config is the root configuration structure for LazyFire.
type Config struct {
	UI UIConfig `mapstructure:"ui"`
}

// UIConfig contains user interface configuration options.
type UIConfig struct {
	Theme ThemeConfig `mapstructure:"theme"`
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

// LoadConfig loads configuration from file or returns defaults.
// It searches for config.yaml in ~/.lazyfire/ and the current directory.
func LoadConfig() (*Config, error) {
	// Default configuration
	config := &Config{
		UI: UIConfig{
			Theme: ThemeConfig{
				ActiveBorderColor:   []string{"cyan"},
				InactiveBorderColor: []string{"default"},
				OptionsTextColor:    []string{"cyan"},
				SelectedLineBgColor: []string{"blue"},
			},
		},
	}

	// Create config directory if it doesn't exist
	home, err := os.UserHomeDir()
	if err != nil {
		return config, nil
	}

	configDir := filepath.Join(home, ".lazyfire")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return config, nil
	}

	// Configure viper to search for config files
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)
	viper.AddConfigPath(".")

	// Read and parse config file if it exists
	if err := viper.ReadInConfig(); err == nil {
		if err := viper.Unmarshal(config); err != nil {
			return config, err
		}
	}

	return config, nil
}
