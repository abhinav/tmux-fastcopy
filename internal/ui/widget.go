package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

//go:generate mockgen -destination mock_widget_test.go -package ui github.com/abhinav/tmux-fastcopy/internal/ui Widget

// Widget is a drawable object that may handle events.
type Widget interface {
	// Draw draws the widget on the supplied view. Widgets do not need to
	// clear the view; the caller will do that for them.
	Draw(views.View)

	// HandleEvent handles the given event, or returns false if the event
	// wasn't meant for it.
	HandleEvent(tcell.Event) (handled bool)
}
