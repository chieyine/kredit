package erp

import (
	"strings"
	"unicode"
)

// Prefix formula-like text, including text preceded by controls or whitespace.
// Numeric money columns are serialized separately and are never modified.
// This is a presentation encoding, not a universal CSV security guarantee:
// recipients should import these columns as text and not strip the prefix.
func spreadsheetText(value string) string {
	significant := strings.TrimLeftFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) || r == '\ufeff' || r == '\u200b' })
	if significant == "" {
		return value
	}
	first := []rune(significant)[0]
	switch first {
	case '=', '+', '-', '@', '＝', '＋', '－', '＠':
		return "'" + value
	}
	// Leading control characters are themselves interpreted specially by
	// some spreadsheet importers. Keep them behind a text prefix as well.
	if len(value) > 0 && (value[0] == '\t' || value[0] == '\r' || value[0] == '\n') {
		return "'" + value
	}
	return value
}
