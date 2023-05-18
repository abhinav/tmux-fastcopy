// Package log provides leveled logging interface.
// The log messages are intended to be user-facing
// similar to the standard library's log package.
package log

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode"
)

// Discard is a logger that discards all its operations.
var Discard = New(io.Discard).WithLevel(discard)

// Level specifies the level of logging.
type Level int

// Supported log levels.
const (
	Debug Level = iota - 1
	Info
	Error
	discard
)

func (l Level) String() string {
	switch l {
	case Debug:
		return "debug"
	case Info:
		return "info"
	case Error:
		return "error"
	default:
		return fmt.Sprintf("%d", int(l))
	}
}

// Logger is a thread-safe logger safe for concurrent use.
type Logger struct {
	w    *lockedWriter
	name string
	lvl  Level
}

// New builds a logger that writes to the given writer.
// The logger defaults to level Info.
func New(w io.Writer) *Logger {
	return &Logger{w: &lockedWriter{W: w}}
}

// WithName builds a new logger with the provided name. The returned logger is
// safe to use concurrently with this logger.
func (l *Logger) WithName(name string) *Logger {
	out := *l
	out.name = name
	return &out
}

// WithLevel buils a new logger that will log messages of the given level or
// higher. The returned logger is safe to use concurrently with this logger.
func (l *Logger) WithLevel(lvl Level) *Logger {
	out := *l
	out.lvl = lvl
	return &out
}

// Level reports the level of the logger. The logger will only log messages of
// this level or higher.
func (l *Logger) Level() Level { return l.lvl }

// Debugf logs messages at the debug level.
func (l *Logger) Debugf(msg string, args ...interface{}) {
	l.Log(Debug, msg, args...)
}

// Infof logs messages at the info level.
func (l *Logger) Infof(msg string, args ...interface{}) {
	l.Log(Info, msg, args...)
}

// Errorf logs messages at the error level.
func (l *Logger) Errorf(msg string, args ...interface{}) {
	l.Log(Error, msg, args...)
}

// Log logs messages at the provided level.
func (l *Logger) Log(level Level, msg string, args ...interface{}) {
	if level < l.lvl {
		return
	}

	var out strings.Builder
	if len(l.name) > 0 {
		out.WriteRune('[')
		out.WriteString(l.name)
		out.WriteString("] ")
	}

	// Ensure a single trailing newline.
	msg = strings.TrimRightFunc(msg, unicode.IsSpace)
	if len(args) > 0 {
		fmt.Fprintf(&out, msg, args...)
	} else {
		out.WriteString(msg)
	}
	out.WriteString("\n")

	_, _ = l.w.WriteString(out.String()) // ignore error
}

type lockedWriter struct {
	mu sync.Mutex
	W  io.Writer
}

func (w *lockedWriter) WriteString(s string) (int, error) {
	w.mu.Lock()
	n, err := io.WriteString(w.W, s)
	w.mu.Unlock()
	return n, err
}
