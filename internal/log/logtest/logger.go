package logtest

import (
	"bytes"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/log"
)

// NewLogger builds a logger at debug level that writes to a testing.T.
func NewLogger(t testing.TB) *log.Logger {
	return log.New(&writer{t}).WithLevel(log.Debug)
}

var _newline = []byte("\n")

type writer struct{ t testing.TB }

func (w *writer) Write(b []byte) (int, error) {
	b = bytes.TrimSuffix(b, _newline)
	w.t.Logf("%s", b)
	return len(b), nil
}
