// Package log provides leveled logging interface.
// The log messages are intended to be user-facing
// similar to the standard library's log package.
package log

import (
	"io"
	"log/slog"

	"github.com/lmittmann/tint"
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
	h := tint.NewHandler(w, &tint.Options{
		Level: lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})

	log := slog.New(h)
	return &Logger{log}
}

// WithName builds a new logger with the provided name. The returned logger is
// safe to use concurrently with this logger.
func (l *Logger) WithName(name string) *Logger {
	out := *l
	out.Logger = l.WithGroup(name)
	return &out
}
