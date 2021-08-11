package log

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestLevels(t *testing.T) {
	t.Parallel()

	t.Run("default level", func(t *testing.T) {
		t.Parallel()

		td.Cmp(t, New(io.Discard).Level(), Info)
	})

	t.Run("info", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		log := New(&buff).WithLevel(Debug)

		log.Debugf("debug")
		log.Infof("info")
		log.Errorf("error")

		td.Cmp(t, buff.String(), unlines("debug", "info", "error"))
	})

	t.Run("info", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		log := New(&buff).WithLevel(Info)

		log.Debugf("debug")
		log.Infof("info")
		log.Errorf("error")

		td.Cmp(t, buff.String(), unlines("info", "error"))
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		log := New(&buff).WithLevel(Error)

		log.Debugf("debug")
		log.Infof("info")
		log.Errorf("error")

		td.Cmp(t, buff.String(), unlines("error"))
	})

	t.Run("discard", func(t *testing.T) {
		t.Parallel()

		var buff bytes.Buffer
		log := New(&buff).WithLevel(discard)

		log.Debugf("debug")
		log.Infof("info")
		log.Errorf("error")

		td.CmpEmpty(t, buff.String())
	})
}

func TestName(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	log := New(&buff).WithName("foo")

	log.Infof("info")
	log.Errorf("error")

	td.Cmp(t, buff.String(), unlines(
		"[foo] info",
		"[foo] error",
	))
}

func TestFormatting(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	log := New(&buff).WithLevel(Debug)

	log.Debugf("level = %v", Debug)
	log.Infof("level = %v", Info)
	log.Errorf("level = %v", Error)

	td.Cmp(t, buff.String(), unlines(
		"level = debug",
		"level = info",
		"level = error",
	))
}

func TestTrailingNewline(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	log := New(&buff)

	log.Infof("foo\n\n")

	td.Cmp(t, buff.String(), unlines("foo"))
}

func unlines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}
