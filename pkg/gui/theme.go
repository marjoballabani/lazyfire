package gui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jesseduffield/gocui"
	"github.com/mballabani/lazyfire/pkg/config"
)

type Theme struct {
	ActiveBorderColor   gocui.Attribute
	InactiveBorderColor gocui.Attribute
	OptionsTextColor    gocui.Attribute
	SelectedLineBgColor gocui.Attribute
}

func NewTheme(cfg config.ThemeConfig) *Theme {
	return &Theme{
		ActiveBorderColor:   parseColor(cfg.ActiveBorderColor),
		InactiveBorderColor: parseColor(cfg.InactiveBorderColor),
		OptionsTextColor:    parseColor(cfg.OptionsTextColor),
		SelectedLineBgColor: parseColor(cfg.SelectedLineBgColor),
	}
}

func parseColor(colorSpec []string) gocui.Attribute {
	if len(colorSpec) == 0 {
		return gocui.ColorDefault
	}

	var attr gocui.Attribute

	for _, spec := range colorSpec {
		spec = strings.ToLower(strings.TrimSpace(spec))

		switch spec {
		case "bold":
			attr |= gocui.AttrBold
		case "underline":
			attr |= gocui.AttrUnderline
		case "reverse":
			attr |= gocui.AttrReverse
		default:
			attr |= parseColorValue(spec)
		}
	}

	return attr
}

func parseColorValue(color string) gocui.Attribute {
	// Handle hex colors
	if strings.HasPrefix(color, "#") {
		return parseHexColor(color)
	}

	// Named colors
	switch color {
	case "default":
		return gocui.ColorDefault
	case "black":
		return gocui.ColorBlack
	case "red":
		return gocui.ColorRed
	case "green":
		return gocui.ColorGreen
	case "yellow":
		return gocui.ColorYellow
	case "blue":
		return gocui.ColorBlue
	case "magenta":
		return gocui.ColorMagenta
	case "cyan":
		return gocui.ColorCyan
	case "white":
		return gocui.ColorWhite
	default:
		// Try parsing as a number (256 color)
		if n, err := strconv.Atoi(color); err == nil && n >= 0 && n < 256 {
			return gocui.Attribute(n) | gocui.AttrIsValidColor
		}
		return gocui.ColorDefault
	}
}

func parseHexColor(hex string) gocui.Attribute {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return gocui.ColorDefault
	}

	r, err := strconv.ParseInt(hex[0:2], 16, 64)
	if err != nil {
		return gocui.ColorDefault
	}
	g, err := strconv.ParseInt(hex[2:4], 16, 64)
	if err != nil {
		return gocui.ColorDefault
	}
	b, err := strconv.ParseInt(hex[4:6], 16, 64)
	if err != nil {
		return gocui.ColorDefault
	}

	return gocui.NewRGBColor(int32(r), int32(g), int32(b))
}

// GetAnsiColorCode returns ANSI escape code for the active border color
func (t *Theme) GetAnsiColorCode() string {
	return attributeToAnsi(t.ActiveBorderColor)
}

func attributeToAnsi(attr gocui.Attribute) string {
	// Check for RGB color (true color)
	if attr&gocui.AttrIsValidColor != 0 {
		// Extract RGB values from attribute
		rgb := uint32(attr & 0xFFFFFF)
		r := (rgb >> 16) & 0xFF
		g := (rgb >> 8) & 0xFF
		b := rgb & 0xFF
		return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
	}

	// Basic colors - check the lower bits
	switch attr & 0xFF {
	case gocui.Attribute(0): // ColorDefault
		return "\033[36m" // Default to cyan
	case gocui.Attribute(1): // ColorBlack
		return "\033[30m"
	case gocui.Attribute(2): // ColorRed
		return "\033[31m"
	case gocui.Attribute(3): // ColorGreen
		return "\033[32m"
	case gocui.Attribute(4): // ColorYellow
		return "\033[33m"
	case gocui.Attribute(5): // ColorBlue
		return "\033[34m"
	case gocui.Attribute(6): // ColorMagenta
		return "\033[35m"
	case gocui.Attribute(7): // ColorCyan
		return "\033[36m"
	case gocui.Attribute(8): // ColorWhite
		return "\033[37m"
	default:
		return "\033[36m" // Default to cyan
	}
}
