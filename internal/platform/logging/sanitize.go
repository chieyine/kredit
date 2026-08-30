package logging

import (
	"net/url"
	"regexp"
	"strings"
)

var sensitiveValuePattern = regexp.MustCompile(`(?i)(bearer\s+|token[=:]|secret[=:]|password[=:]|otp[=:]|pin[=:]|bvn[=:]|nin[=:]|account(?:_number)?[=:])[^\s,;]+`)
var credentialURLPattern = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://[^:/\s]+:)[^@/\s]+@`)

// Redact removes credentials and restricted identifiers from values that may be
// emitted to logs. Logging is an operational diagnostic, not a data export.
func Redact(value string) string {
	if value == "" {
		return ""
	}
	value = sensitiveValuePattern.ReplaceAllString(value, "$1[redacted]")
	value = credentialURLPattern.ReplaceAllString(value, "${1}[redacted]@")
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		return r
	}, value)
}

// SafePath preserves route shape while removing query strings and opaque
// identifiers from known sensitive paths. It is safe to use in access logs.
func SafePath(rawPath string) string {
	parsed, err := url.Parse(rawPath)
	if err != nil {
		return "[redacted]"
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	segments := strings.Split(path, "/")
	sensitiveParent := map[string]bool{
		"buyer-invitations": true,
		"documents":         true,
		"webhooks":          true,
		"uploads":           true,
	}
	for index := range segments {
		if index > 0 && sensitiveParent[segments[index-1]] && segments[index] != "" {
			segments[index] = "[redacted]"
		}
	}
	return strings.Join(segments, "/")
}

// SafeAttributes returns a copy suitable for slog. Keys associated with
// secrets or personal data are replaced rather than merely value-redacted.
func SafeAttributes(attributes map[string]string) map[string]string {
	result := make(map[string]string, len(attributes))
	for key, value := range attributes {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "otp") || strings.Contains(lower, "pin") || strings.Contains(lower, "bvn") || strings.Contains(lower, "nin") || strings.Contains(lower, "phone") || strings.Contains(lower, "email") || strings.Contains(lower, "account") {
			result[key] = "[redacted]"
			continue
		}
		result[key] = Redact(value)
	}
	return result
}
