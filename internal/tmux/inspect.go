package tmux

import (
	"github.com/abhinav/tmux-fastcopy/internal/stringobj"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxfmt"
)

// PaneMode specifies the mode in which the pane is.
type PaneMode string

const (
	// NormalMode specifies that the tmux pane is in normal mode, at the
	// bottom of the screen.
	NormalMode PaneMode = "normal-mode"

	// CopyMode indicates that the tmux pane is in copy mode, and may be
	// scrolled up.
	CopyMode PaneMode = "copy-mode"
)

// PaneInfo reports information about a tmux pane.
type PaneInfo struct {
	ID               string
	WindowID         string
	ClientName       string
	Width, Height    int
	Mode             PaneMode
	CursorX, CursorY int
	ScrollPosition   int
}

func (i *PaneInfo) String() string {
	var b stringobj.Builder
	b.Put("id", i.ID)
	b.Put("windowID", i.WindowID)
	b.Put("clientName", i.ClientName)
	b.Put("width", i.Width)
	b.Put("width", i.Height)
	b.Put("mode", i.Mode)
	b.Put("cursorX", i.CursorX)
	b.Put("cursorY", i.CursorY)
	b.Put("scrollPosition", i.ScrollPosition)
	return b.String()
}

var (
	_paneID     = tmuxfmt.Var("pane_id")
	_paneWidth  = tmuxfmt.Var("pane_width")
	_paneHeight = tmuxfmt.Var("pane_height")
	_paneMode   = tmuxfmt.Ternary{
		Cond: tmuxfmt.Var("pane_in_mode"),
		Then: tmuxfmt.Var("pane_mode"),
		Else: tmuxfmt.String("normal-mode"),
	}
	_paneInCopyMode = tmuxfmt.Binary{
		LHS: tmuxfmt.Var("pane_mode"),
		Op:  tmuxfmt.Equals,
		RHS: tmuxfmt.String("copy-mode"),
	}
	_paneCursorX = tmuxfmt.Ternary{
		Cond: _paneInCopyMode,
		Then: tmuxfmt.Var("copy_cursor_x"),
		Else: tmuxfmt.Var("cursor_x"),
	}
	_paneCursorY = tmuxfmt.Ternary{
		Cond: _paneInCopyMode,
		Then: tmuxfmt.Var("copy_cursor_y"),
		Else: tmuxfmt.Var("cursor_y"),
	}
	_paneScrollPosition = tmuxfmt.Ternary{
		Cond: _paneInCopyMode,
		Then: tmuxfmt.Var("scroll_position"),
		Else: tmuxfmt.Int(0),
	}
	_windowID   = tmuxfmt.Var("window_id")
	_clientName = tmuxfmt.Var("client_name")
)

// InspectPane inspects a tmux pane and reports information about it. The
// argument identifies the pane we want to inspect, defaulting to the current
// pane if none is specified.
func InspectPane(driver Driver, identifier string) (*PaneInfo, error) {
	var (
		info PaneInfo
		fc   tmuxfmt.Capturer
	)
	fc.StringVar(&info.ID, _paneID)
	fc.StringVar(&info.WindowID, _windowID)
	fc.StringVar(&info.ClientName, _clientName)
	fc.IntVar(&info.Width, _paneWidth)
	fc.IntVar(&info.Height, _paneHeight)
	fc.StringVar((*string)(&info.Mode), _paneMode)
	fc.IntVar(&info.CursorX, _paneCursorX)
	fc.IntVar(&info.CursorY, _paneCursorY)
	fc.IntVar(&info.ScrollPosition, _paneScrollPosition)

	msg, parse := fc.Prepare()
	out, err := driver.DisplayMessage(DisplayMessageRequest{
		Pane:    identifier,
		Message: msg,
	})
	if err == nil {
		err = parse(out)
	}
	return &info, err
}
