// Package log provides leveled logging interface.
// The log messages are intended to be user-facing
// similar to the standard library's log package.
package log

import (
	"io"
	"log/slog"
)

// Level specifies the level of logging.
type Level = slog.Level

// Supported log levels.
const (
	Debug = slog.LevelDebug
	Info  = slog.LevelInfo
	Error = slog.LevelError
)

type Logger struct{ *slog.Logger }

// New builds a logger that writes to the given writer.
// The logger defaults to level Info.
func New(w io.Writer, lvl Level) *Logger {
	log := slog.New(&handler{
		W:     w,
		Level: lvl,
	})
	return &Logger{log}
}

// WithName builds a new logger with the provided name. The returned logger is
// safe to use concurrently with this logger.
func (l *Logger) WithName(name string) *Logger {
	out := *l
	out.Logger = l.WithGroup(name)
	return &out
}
