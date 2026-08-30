package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

func New() *slog.Logger {
	return slog.New(NewSanitizingHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
}

type sanitizingHandler struct{ next slog.Handler }

func NewSanitizingHandler(next slog.Handler) slog.Handler { return &sanitizingHandler{next: next} }

func (h *sanitizingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *sanitizingHandler) Handle(ctx context.Context, record slog.Record) error {
	attributes := make([]slog.Attr, 0, record.NumAttrs())
	record.Attrs(func(attribute slog.Attr) bool {
		attributes = append(attributes, sanitizeAttr(attribute))
		return true
	})
	safeRecord := slog.NewRecord(record.Time, record.Level, Redact(record.Message), record.PC)
	safeRecord.AddAttrs(attributes...)
	return h.next.Handle(ctx, safeRecord)
}

func (h *sanitizingHandler) WithAttrs(attributes []slog.Attr) slog.Handler {
	result := make([]slog.Attr, len(attributes))
	for index, attribute := range attributes {
		result[index] = sanitizeAttr(attribute)
	}
	return &sanitizingHandler{next: h.next.WithAttrs(result)}
}

func (h *sanitizingHandler) WithGroup(name string) slog.Handler {
	return &sanitizingHandler{next: h.next.WithGroup(name)}
}

func sanitizeAttr(attribute slog.Attr) slog.Attr {
	lower := strings.ToLower(attribute.Key)
	if lower == "error" || lower == "panic" {
		if err, ok := attribute.Value.Any().(error); ok {
			return slog.String(attribute.Key, SafeError(err))
		}
		return slog.String(attribute.Key, Redact(attribute.Value.String()))
	}
	if sensitiveMetadataKey(lower) {
		return slog.String(attribute.Key, "[redacted]")
	}
	if attribute.Value.Kind() == slog.KindString {
		return slog.String(attribute.Key, Redact(attribute.Value.String()))
	}
	if attribute.Value.Kind() == slog.KindAny {
		if err, ok := attribute.Value.Any().(error); ok {
			return slog.String(attribute.Key, SafeError(err))
		}
	}
	return attribute
}

func sensitiveMetadataKey(key string) bool {
	for _, fragment := range []string{"token", "secret", "password", "otp", "pin", "bvn", "nin", "phone", "email", "account", "cookie", "authorization"} {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

// SafeError preserves an error-shaped value for slog while removing internal
// connection details, credentials and provider payload fragments.
func SafeError(err error) string {
	if err == nil {
		return ""
	}
	detail := Redact(err.Error())
	lower := strings.ToLower(detail)
	for _, marker := range []string{"postgres", "pgx", "provider", "response body", "webhook body", "authorization", "cookie", "secret", "password", "token", "stack trace"} {
		if strings.Contains(lower, marker) {
			return "operation failed"
		}
	}
	if len(detail) > 512 {
		return detail[:512]
	}
	return detail
}
