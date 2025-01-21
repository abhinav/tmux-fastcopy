// Package logtest provides a logger that can write to a testing.T.
package logtest

import (
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/log"
	"go.abhg.dev/io/ioutil"
)

// NewLogger builds a logger at debug level that writes to a testing.T.
func NewLogger(t testing.TB) *log.Logger {
	return log.New(ioutil.TestLogWriter(t, "")).WithLevel(log.Debug)
}
