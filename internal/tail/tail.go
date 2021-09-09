// Package tail provides support for tailing an io.Reader that isn't yet done
// filling up.
package tail

import (
	"errors"
	"io"
	"io/fs"
	"time"

	"github.com/benbjohnson/clock"
)

const (
	_defaultDelay      = 100 * time.Millisecond
	_defaultBufferSize = 32 * 1024 // 32kB
)

// Tee copies text from source to destination until the user closes the source
// or calls Tee.Stop.
type Tee struct {
	W io.Writer // destination (required)
	R io.Reader // source (required)

	// Maximum delay between retries. If the end of the source is reached,
	// we'll wait up to this much time before trying again. Defaults to 100
	// milliseconds.
	Delay time.Duration

	// Size of the copy buffer. Defaults to 32kB.
	BufferSize int

	Clock clock.Clock

	err        error
	buffer     []byte
	quit, done chan struct{}
}

// Start begins tailing the source and copying blobs into destination until an
// error is encountered, source runs out, or Stop is called. If source reaches
// EOF but is not yet closed, Tee will try again after some delay (configurable
// via the Delay parameter).
//
// Start returns immediately.
func (t *Tee) Start() {
	if t.Delay == 0 {
		t.Delay = _defaultDelay
	}
	if t.BufferSize == 0 {
		t.BufferSize = _defaultBufferSize
	}
	if t.Clock == nil {
		t.Clock = clock.New()
	}

	t.buffer = make([]byte, t.BufferSize)
	t.quit = make(chan struct{})
	t.done = make(chan struct{})

	go t.run()
}

// Stop tells Tee to stop copying text. It blocks until it has cleaned up the
// background job. Returns errors encountered during run, if any.
//
// If this freezes, make sure you closed the underlying file.
func (t *Tee) Stop() error {
	close(t.quit)

	return t.Wait()
}

// Wait waits until the tee stops from an error or from Stop being called.
// Returns the error, if any.
func (t *Tee) Wait() error {
	<-t.done
	return t.err
}

func (t *Tee) run() {
	defer close(t.done)

	ticker := t.Clock.Ticker(t.Delay)
	defer ticker.Stop()

	for {
		n, err := io.CopyBuffer(t.W, t.R, t.buffer)
		if err == nil && n > 0 {
			// There are more bytes still to read.
			continue
		}

		switch {
		case errors.Is(err, fs.ErrClosed):
			// File is closed. No new logs are expected.
			return

		case err == nil || errors.Is(err, io.EOF):
			// There were no more bytes left to copy. Wait for quit
			// or up to the specified delay and try again.
			select {
			case <-t.quit:
				return
			case <-ticker.C:
			}

		default:
			// Something went wrong. Record and die.
			t.err = err
			return
		}
	}
}
