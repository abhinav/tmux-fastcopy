package log

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevels(t *testing.T) {
	t.Parallel()

	t.Run("default level", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, Info, New(io.Discard).Level())
	})

	t.Run("info", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		log := New(&buff).WithLevel(Debug)

		log.Debugf("debug")
		log.Infof("info")
		log.Errorf("error")

		assert.Equal(t, unlines("debug", "info", "error"), buff.String())
	})

	t.Run("info", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		log := New(&buff).WithLevel(Info)

		log.Debugf("debug")
		log.Infof("info")
		log.Errorf("error")

		assert.Equal(t, unlines("info", "error"), buff.String())
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		log := New(&buff).WithLevel(Error)

		log.Debugf("debug")
		log.Infof("info")
		log.Errorf("error")

		assert.Equal(t, unlines("error"), buff.String())
	})

	t.Run("discard", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		log := New(&buff).WithLevel(discard)

		log.Debugf("debug")
		log.Infof("info")
		log.Errorf("error")

		assert.Empty(t, buff.String())
	})
}

func TestName(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	log := New(&buff).WithName("foo")

	log.Infof("info")
	log.Errorf("error")

	assert.Equal(t, unlines(
		"[foo] info",
		"[foo] error",
	), buff.String())
}

func TestFormatting(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	log := New(&buff).WithLevel(Debug)

	log.Debugf("level = %v", Debug)
	log.Infof("level = %v", Info)
	log.Errorf("level = %v", Error)
	log.Errorf("level = %v", discard)

	assert.Equal(t, unlines(
		"level = debug",
		"level = info",
		"level = error",
		"level = 2",
	), buff.String())
}

func TestTrailingNewline(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	log := New(&buff)

	log.Infof("foo\n\n")

	assert.Equal(t, unlines("foo"), buff.String())
}

func unlines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}
