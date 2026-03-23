package gui

import (
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Terminal256 formatter that outputs ANSI codes
type ansiFormatter struct{}

// Format outputs tokens with ANSI color codes
func (f *ansiFormatter) Format(tokens []chroma.Token) string {
	var result strings.Builder
	for _, token := range tokens {
		color := tokenColor(token.Type)
		if color != "" {
			result.WriteString(color)
			result.WriteString(token.Value)
			result.WriteString("\033[0m")
		} else {
			result.WriteString(token.Value)
		}
	}
	return result.String()
}

// tokenColor returns ANSI color code for token type
func tokenColor(t chroma.TokenType) string {
	switch {
	case t == chroma.NameTag || t == chroma.NameAttribute:
		return "\033[36m" // Cyan for keys
	case t == chroma.LiteralString || t == chroma.LiteralStringSingle || t == chroma.LiteralStringDouble:
		return "\033[32m" // Green for strings
	case t == chroma.LiteralNumber || t == chroma.LiteralNumberFloat || t == chroma.LiteralNumberInteger:
		return "\033[33m" // Yellow for numbers
	case t == chroma.KeywordConstant: // true, false, null
		return "\033[35m" // Magenta for booleans/null
	case t == chroma.Punctuation:
		return "\033[90m" // Gray for brackets/punctuation
	default:
		return ""
	}
}

// colorizeJSON adds ANSI color codes to JSON string for terminal display
// Uses chroma lexer for fast, accurate tokenization
func colorizeJSON(jsonStr string) string {
	lexer := lexers.Get("json")
	if lexer == nil {
		return jsonStr // Fallback to plain
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, jsonStr)
	if err != nil {
		return jsonStr
	}

	tokens := iterator.Tokens()
	formatter := &ansiFormatter{}
	return formatter.Format(tokens)
}

// colorizeLine applies syntax highlighting to a single line of JSON
// Used for incremental colorization
func colorizeLine(line string) string {
	// For single lines, use the full colorizer
	// Chroma handles partial JSON gracefully
	return colorizeJSON(line)
}

// annotateTimestamps appends human-readable timestamp comments to colorized JSON.
// rawJSON is used to detect timestamps; colorized is the ANSI-highlighted version.
// Both must have the same number of lines.
func annotateTimestamps(rawJSON, colorized string) string {
	rawLines := strings.Split(rawJSON, "\n")
	colorLines := strings.Split(colorized, "\n")

	for i, rawLine := range rawLines {
		if i >= len(colorLines) {
			break
		}
		// Look for quoted ISO timestamp strings
		if idx := strings.Index(rawLine, `"`); idx >= 0 {
			rest := rawLine[idx+1:]
			endIdx := strings.Index(rest, `"`)
			if endIdx > 0 {
				val := rest[:endIdx]
				if t, err := time.Parse(time.RFC3339Nano, val); err == nil {
					humanized := t.Local().Format("Jan 2, 2006 3:04:05 PM")
					colorLines[i] = colorLines[i] + "  \033[90m// " + humanized + "\033[0m"
				} else if t, err := time.Parse(time.RFC3339, val); err == nil {
					humanized := t.Local().Format("Jan 2, 2006 3:04:05 PM")
					colorLines[i] = colorLines[i] + "  \033[90m// " + humanized + "\033[0m"
				}
			}
		}
	}
	return strings.Join(colorLines, "\n")
}

// Ensure styles package is used (prevents unused import)
var _ = styles.Fallback
