package log

import (
	"context"
	"log/slog"
)

// Discard is a logger that discards all its operations.
var Discard = &Logger{slog.New(&discardHandler{})}

type discardHandler struct{}

func (*discardHandler) Enabled(context.Context, slog.Level) bool {
	return false
}

func (*discardHandler) Handle(context.Context, slog.Record) error {
	return nil
}

func (h *discardHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *discardHandler) WithGroup(name string) slog.Handler {
	return h
}
