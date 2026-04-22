package ui

import (
	"github.com/gdamore/tcell/v3"
	"github.com/rivo/uniseg"
)

// DrawText draws a string on the provided view at the specified position.
// Returns the new position, after having drawn the text, making it possible to
// continue drawing at the last written position.
//
//	pos = DrawText("foo\nb", style, view, pos)
//	pos = DrawText("ar", style, view, pos)
//
// Text that bleeds outside the bounds of the view is ignored.
func DrawText(s string, style tcell.Style, view View, pos Pos) Pos {
	if len(s) == 0 {
		return pos
	}

	w, h := view.Size()
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		s := g.Str()
		if pos.X >= w || s == "\n" {
			pos.Y++
			pos.X = 0
		}

		if pos.Y >= h {
			return pos
		}

		if s != "\n" {
			_, width := view.Put(pos.X, pos.Y, s, style)
			pos.X += width
		}
	}

	return pos
}
