package ui

import (
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestPos(t *testing.T) {
	t.Parallel()

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		var p Pos

		x, y := p.Get()
		td.Cmp(t, x, 0, "X")
		td.Cmp(t, y, 0, "y")

		td.Cmp(t, p.String(), "(0, 0)")
	})

	t.Run("zero", func(t *testing.T) {
		t.Parallel()

		p := Pos{5, 6}

		x, y := p.Get()
		td.Cmp(t, x, 5, "X")
		td.Cmp(t, y, 6, "y")

		td.Cmp(t, p.String(), "(5, 6)")
	})
}
