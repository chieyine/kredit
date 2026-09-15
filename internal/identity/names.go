package identity

import (
	"sort"
	"strings"
)

// NamesMatch tolerates name order and whitespace, but never missing names or
// partial matches. Ambiguous bank names require corrected provider evidence.
func NamesMatch(a, b string) bool {
	normalize := func(s string) string {
		parts := strings.Fields(strings.ToUpper(s))
		sort.Strings(parts)
		return strings.Join(parts, " ")
	}
	a, b = normalize(a), normalize(b)
	return a != "" && a == b
}
