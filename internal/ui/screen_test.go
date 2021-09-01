package ui

import (
	"testing"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/require"
)

func NewTestScreen(t testing.TB, w, h int) tcell.SimulationScreen {
	scr := tcell.NewSimulationScreen("")
	require.NoError(t, scr.Init())
	t.Cleanup(scr.Fini)
	scr.SetSize(w, h)
	return scr
}
