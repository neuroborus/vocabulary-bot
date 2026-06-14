package logging

import (
	"context"
	"log/slog"
)

type sanitizingHandler struct {
	inner slog.Handler
}

func NewSanitizingHandler(inner slog.Handler) slog.Handler {
	if inner == nil {
		return nil
	}

	return &sanitizingHandler{inner: inner}
}

func (h *sanitizingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *sanitizingHandler) Handle(ctx context.Context, record slog.Record) error {
	cloned := slog.NewRecord(record.Time, record.Level, SanitizeString(record.Message), record.PC)
	record.Attrs(func(attr slog.Attr) bool {
		cloned.AddAttrs(sanitizeAttr(attr))
		return true
	})

	return h.inner.Handle(ctx, cloned)
}

func (h *sanitizingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &sanitizingHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *sanitizingHandler) WithGroup(name string) slog.Handler {
	return &sanitizingHandler{inner: h.inner.WithGroup(name)}
}

func sanitizeAttr(attr slog.Attr) slog.Attr {
	if attr.Equal(slog.Attr{}) {
		return attr
	}

	if attr.Value.Kind() == slog.KindGroup {
		groupAttrs := attr.Value.Group()
		sanitized := make([]any, 0, len(groupAttrs))
		for _, groupAttr := range groupAttrs {
			sanitized = append(sanitized, sanitizeAttr(groupAttr))
		}

		return slog.Group(attr.Key, sanitized...)
	}

	if attr.Value.Kind() == slog.KindString {
		return slog.String(attr.Key, SanitizeString(attr.Value.String()))
	}

	if attr.Value.Kind() == slog.KindLogValuer {
		return slog.String(attr.Key, SanitizeString(attr.Value.String()))
	}

	if attr.Value.Kind() == slog.KindAny {
		if text, ok := attr.Value.Any().(string); ok {
			return slog.String(attr.Key, SanitizeString(text))
		}
	}

	return attr
}
