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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	tee := Tee{
		W:     &buff,
		R:     r,
		Clock: clock,
	}
	tee.Start()
	defer func() {
		assert.NoError(t, r.Close())
		assert.NoError(t, tee.Stop())
	}()

	w, err := os.OpenFile(r.Name(), os.O_WRONLY, 0644)
	if !assert.NoError(t, err) {
		return
	}
	defer func() { assert.NoError(t, w.Close()) }()

	t.Run("empty", func(t *testing.T) {
		assert.Empty(t, buff.String())
	})

	t.Run("write", func(t *testing.T) {
		defer buff.Reset()

		io.WriteString(w, "hello")
		clock.Add(_defaultDelay)
		assert.Equal(t, "hello", buff.String())
	})

	t.Run("write delayed", func(t *testing.T) {
		defer buff.Reset()

		for i := 0; i < 10; i++ {
			clock.Add(_defaultDelay * 10)
			assert.Empty(t, buff.String())
		}

		io.WriteString(w, "world")
		clock.Add(_defaultDelay)
		assert.Equal(t, "world", buff.String())
	})
}

func TestTeeError(t *testing.T) {
	t.Parallel()

	var buff lockedBuffer
	defer func() { assert.Empty(t, buff.String()) }()

	r := iotest.ErrReader(errors.New("great sadness"))
	tee := Tee{
		W: &buff,
		R: ioutil.NopCloser(r),
	}
	tee.Start()

	err := tee.Stop()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "great sadness")
}

func TestTeeClosed(t *testing.T) {
	t.Parallel()

	var buff lockedBuffer
	defer func() { assert.Empty(t, buff.String()) }()

	r, err := ioutil.TempFile(t.TempDir(), "file")
	require.NoError(t, err)

	tee := Tee{
		W: &buff,
		R: r,
	}
	tee.Start()
	defer func() {
		assert.NoError(t, tee.Stop())
	}()

	assert.NoError(t, r.Close())
}
