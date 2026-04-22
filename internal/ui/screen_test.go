package ui

import (
	"testing"

	tcell "github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/stretchr/testify/require"
)

func NewTestScreen(t testing.TB, w, h int) (vt.MockTerm, tcell.Screen, func()) {
	t.Helper()

	term := vt.NewMockTerm(vt.MockOptSize{X: vt.Col(w), Y: vt.Row(h)})
	scr, err := tcell.NewTerminfoScreenFromTty(term)
	require.NoError(t, err)
	require.NoError(t, scr.Init())
	return term, scr, scr.Fini
}
