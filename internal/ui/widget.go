package ui

import (
	"github.com/gdamore/tcell/v3"
)

// View is the minimal drawing surface widgets need from tcell.
type View interface {
	Size() (int, int)
	Put(x int, y int, str string, style tcell.Style) (string, int)
}

//go:generate mockgen -destination mock_widget_test.go -package ui github.com/abhinav/tmux-fastcopy/internal/ui Widget

// Widget is a drawable object that may handle events.
type Widget interface {
	// Draw draws the widget on the supplied view. Widgets do not need to
	// clear the view; the caller will do that for them.
	Draw(View)

	// HandleEvent handles the given event, or returns false if the event
	// wasn't meant for it.
	HandleEvent(tcell.Event) (handled bool)
}
