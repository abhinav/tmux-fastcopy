// Package iotest provides IO-related testing utilities.
package iotest

import (
	"bytes"
	"io"
)

var _newline = []byte("\n")

// Logger is the destination for messages from the Writer.
// It is satisfied by *testing.T and *testing.B.
type Logger interface {
	Logf(format string, args ...interface{})
}

// Writer builds an io.Writer that writes to the given testing.TB.
func Writer(t Logger) io.Writer {
	return &writer{t}
}

type writer struct{ t Logger }

func (w *writer) Write(b []byte) (int, error) {
	b = bytes.TrimSuffix(b, _newline)
	w.t.Logf("%s", b)
	return len(b), nil
}
