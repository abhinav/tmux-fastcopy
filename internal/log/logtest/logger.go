package logtest

import (
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/iotest"
	"github.com/abhinav/tmux-fastcopy/internal/log"
)

// NewLogger builds a logger at debug level that writes to a testing.T.
func NewLogger(t testing.TB) *log.Logger {
	return log.New(iotest.Writer(t)).WithLevel(log.Debug)
}
