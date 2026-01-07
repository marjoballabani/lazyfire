package gui

import (
	"regexp"
	"strings"
)

// ANSI color codes for JSON syntax highlighting
const (
	colorReset   = "\033[0m"
	colorKey     = "\033[36m"  // Cyan for keys
	colorString  = "\033[32m"  // Green for string values
	colorNumber  = "\033[33m"  // Yellow for numbers
	colorBool    = "\033[35m"  // Magenta for booleans
	colorNull    = "\033[31m"  // Red for null
	colorBracket = "\033[90m"  // Gray for brackets
)

// Precompiled regex pattern for JSON key detection
var jsonKeyPattern = regexp.MustCompile(`"([^"\\]|\\.)*"\s*:`)

// colorizeJSON adds ANSI color codes to JSON string for terminal display
func colorizeJSON(jsonStr string) string {
	var result strings.Builder
	lines := strings.Split(jsonStr, "\n")

	for i, line := range lines {
		colored := colorizeLine(line)
		result.WriteString(colored)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// colorizeLine applies syntax highlighting to a single line of JSON
func colorizeLine(line string) string {
	// Check for key-value patterns
	if match := jsonKeyPattern.FindStringIndex(line); match != nil {
		keyEnd := match[1]
		key := line[match[0]:keyEnd]
		rest := line[keyEnd:]

		// Colorize the key
		coloredKey := colorKey + key + colorReset

		// Colorize the value
		coloredValue := colorizeValue(rest)

		return line[:match[0]] + coloredKey + coloredValue
	}

	// No key found, just colorize brackets
	return colorizeBrackets(line)
}

// colorizeValue applies color to JSON values
func colorizeValue(s string) string {
	s = strings.TrimSpace(s)

	// Check for string value
	if strings.HasPrefix(s, `"`) {
		return " " + colorString + s + colorReset
	}

	// Check for number
	if len(s) > 0 && (s[0] == '-' || (s[0] >= '0' && s[0] <= '9')) {
		// Find end of number
		end := 0
		for end < len(s) && (s[end] == '-' || s[end] == '.' || s[end] == 'e' || s[end] == 'E' || s[end] == '+' || (s[end] >= '0' && s[end] <= '9')) {
			end++
		}
		if end > 0 {
			return " " + colorNumber + s[:end] + colorReset + s[end:]
		}
	}

	// Check for boolean
	if strings.HasPrefix(s, "true") {
		return " " + colorBool + "true" + colorReset + s[4:]
	}
	if strings.HasPrefix(s, "false") {
		return " " + colorBool + "false" + colorReset + s[5:]
	}

	// Check for null
	if strings.HasPrefix(s, "null") {
		return " " + colorNull + "null" + colorReset + s[4:]
	}

	// Check for array/object start
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
		return " " + colorizeBrackets(s)
	}

	return " " + s
}

// colorizeBrackets adds color to brackets and braces
func colorizeBrackets(s string) string {
	var result strings.Builder
	for _, ch := range s {
		switch ch {
		case '{', '}', '[', ']':
			result.WriteString(colorBracket)
			result.WriteRune(ch)
			result.WriteString(colorReset)
		default:
			result.WriteRune(ch)
		}
	}
	return result.String()
}
