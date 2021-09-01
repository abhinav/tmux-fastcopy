package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPos(t *testing.T) {
	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		var p Pos

		x, y := p.Get()
		assert.Equal(t, 0, x, "X")
		assert.Equal(t, 0, y, "y")

		assert.Equal(t, "(0, 0)", p.String())
	})

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		p := Pos{5, 6}

		x, y := p.Get()
		assert.Equal(t, 5, x, "X")
		assert.Equal(t, 6, y, "y")

		assert.Equal(t, "(5, 6)", p.String())
	})
}
