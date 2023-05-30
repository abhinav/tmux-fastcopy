package ui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:paralleltest // shared state between subtests
func TestAnnotatedText(t *testing.T) {
	t.Parallel()

	const (
		W = 3
		H = 3
	)

	normal := tcell.StyleDefault
	highlighted := tcell.StyleDefault.Foreground(tcell.ColorRed)

	scr := NewTestScreen(t, W, H)
	at := AnnotatedText{
		Text:  "foo\nbar\nbaz",
		Style: normal,
	}

	matchScreen := func(t *testing.T, want ...tcell.SimCell) {
		require.Len(t, want, W*H, "invalid test: not enough cells")

		t.Helper()

		got, w, h := scr.GetContents()
		assert.Equal(t, W, w)
		assert.Equal(t, H, h)
		assert.Equal(t, want, got)
	}

	// n generates a normal cell
	n := func(r rune) tcell.SimCell {
		return tcell.SimCell{
			Bytes: []byte(string(r)),
			Style: normal,
			Runes: []rune{r},
		}
	}

	// h generates a highlighted rune.
	h := func(r rune) tcell.SimCell {
		return tcell.SimCell{
			Bytes: []byte(string(r)),
			Style: highlighted,
			Runes: []rune{r},
		}
	}

	t.Run("no annotations", func(t *testing.T) {
		defer scr.Clear()
		defer at.SetAnnotations()

		at.Draw(scr)
		scr.Show()

		matchScreen(t,
			n('f'), n('o'), n('o'),
			n('b'), n('a'), n('r'),
			n('b'), n('a'), n('z'),
		)
	})

	t.Run("empty annotation", func(t *testing.T) {
		defer scr.Clear()
		defer at.SetAnnotations()

		at.SetAnnotations(OverlayTextAnnotation{})

		at.Draw(scr)
		scr.Show()

		matchScreen(t,
			n('f'), n('o'), n('o'),
			n('b'), n('a'), n('r'),
			n('b'), n('a'), n('z'),
		)
	})

	t.Run("style", func(t *testing.T) {
		defer scr.Clear()
		defer at.SetAnnotations()

		at.SetAnnotations(
			StyleTextAnnotation{Style: highlighted, Offset: 4, Length: 1}, // <b>ar
			StyleTextAnnotation{Style: highlighted, Offset: 8, Length: 2}, // <ba>z
			StyleTextAnnotation{Style: highlighted, Offset: 1, Length: 2}, // f<oo>
		)

		// +---+---+---+
		// | 0 |*1*|*2*| 3
		// +---+---+---+
		// |*4*| 5 | 6 | 7
		// +---+---+---+
		// |*8*|*9*|10 | 11
		// +---+---+---+

		at.Draw(scr)
		scr.Show()

		matchScreen(t,
			n('f'), h('o'), h('o'),
			h('b'), n('a'), n('r'),
			h('b'), h('a'), n('z'),
		)
	})

	t.Run("overlay", func(t *testing.T) {
		defer scr.Clear()
		defer at.SetAnnotations()

		at.SetAnnotations(
			OverlayTextAnnotation{Overlay: "a", Offset: 1},
			OverlayTextAnnotation{Overlay: "b", Style: highlighted},
		)

		at.Draw(scr)
		scr.Show()

		matchScreen(t,
			h('b'), n('a'), n('o'),
			n('b'), n('a'), n('r'),
			n('b'), n('a'), n('z'),
		)
	})

	t.Run("overlapping", func(t *testing.T) {
		defer scr.Clear()
		defer at.SetAnnotations()

		at.SetAnnotations(
			OverlayTextAnnotation{Overlay: "abc"},
			OverlayTextAnnotation{Overlay: "de", Offset: 1}, // ignored
		)

		at.Draw(scr)
		scr.Show()

		matchScreen(t,
			n('a'), n('b'), n('c'),
			n('b'), n('a'), n('r'),
			n('b'), n('a'), n('z'),
		)
	})

	t.Run("unknown annotation", func(t *testing.T) {
		defer scr.Clear()
		defer at.SetAnnotations()

		var foo struct{ StyleTextAnnotation }
		foo.Length = 3
		at.SetAnnotations(foo)

		assert.Panics(t, func() {
			at.Draw(scr)
		})
	})
}
