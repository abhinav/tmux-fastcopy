package ui

import (
	"testing"

	tcell "github.com/gdamore/tcell/v2"
)

func NewTestScreen(t testing.TB, w, h int) tcell.SimulationScreen {
	scr := tcell.NewSimulationScreen("")
	if err := scr.Init(); err != nil {
		t.Fatalf("cannot initalize screen: %v", err)
	}
	t.Cleanup(scr.Fini)
	scr.SetSize(w, h)
	return scr
}
