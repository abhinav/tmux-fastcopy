package tail

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/benbjohnson/clock"
	"github.com/maxatome/go-testdeep/td"
)

type lockedBuffer struct {
	mu   sync.RWMutex
	buff bytes.Buffer
}

func (b *lockedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buff.Write(data)
}

func (b *lockedBuffer) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buff.Reset()
}

func (b *lockedBuffer) String() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.buff.String()
}

func TestTee(t *testing.T) {
	t.Parallel()

	clock := clock.NewMock()

	var buff lockedBuffer
	r, err := ioutil.TempFile(t.TempDir(), "file")
	if !td.CmpNoError(t, err) {
		return
	}

	tee := Tee{
		W:     &buff,
		R:     r,
		Clock: clock,
	}
	tee.Start()
	defer func() {
		td.CmpNoError(t, r.Close())
		td.CmpNoError(t, tee.Stop())
	}()

	w, err := os.OpenFile(r.Name(), os.O_WRONLY, 0644)
	if !td.CmpNoError(t, err) {
		return
	}
	defer func() { td.CmpNoError(t, w.Close()) }()

	t.Run("empty", func(t *testing.T) {
		td.CmpEmpty(t, buff.String())
	})

	t.Run("write", func(t *testing.T) {
		defer buff.Reset()

		io.WriteString(w, "hello")
		clock.Add(_defaultDelay)
		td.Cmp(t, buff.String(), "hello")
	})

	t.Run("write delayed", func(t *testing.T) {
		defer buff.Reset()

		for i := 0; i < 10; i++ {
			clock.Add(_defaultDelay * 10)
			td.CmpEmpty(t, buff.String())
		}

		io.WriteString(w, "world")
		clock.Add(_defaultDelay)
		td.Cmp(t, buff.String(), "world")
	})
}

func TestTeeError(t *testing.T) {
	t.Parallel()

	var buff lockedBuffer
	defer func() { td.CmpEmpty(t, buff.String()) }()

	r := iotest.ErrReader(errors.New("great sadness"))
	tee := Tee{
		W: &buff,
		R: ioutil.NopCloser(r),
	}
	tee.Start()

	err := tee.Stop()
	td.CmpError(t, err)
	td.CmpContains(t, err.Error(), "great sadness")
}

func TestTeeClosed(t *testing.T) {
	t.Parallel()

	var buff lockedBuffer
	defer func() { td.CmpEmpty(t, buff.String()) }()

	r, err := ioutil.TempFile(t.TempDir(), "file")
	if !td.CmpNoError(t, err) {
		return
	}

	tee := Tee{
		W: &buff,
		R: r,
	}
	tee.Start()
	defer func() {
		td.CmpNoError(t, tee.Stop())
	}()

	td.CmpNoError(t, r.Close())
}
