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
	ID             string
	WindowID       string
	Width, Height  int
	Mode           PaneMode
	ScrollPosition int
	WindowZoomed   bool

	// Current path of the pane, if available.
	CurrentPath string
}

func (i *PaneInfo) String() string {
	var b stringobj.Builder
	b.Put("id", i.ID)
	b.Put("windowID", i.WindowID)
	b.Put("width", i.Width)
	b.Put("height", i.Height)
	b.Put("mode", i.Mode)
	b.Put("scrollPosition", i.ScrollPosition)
	b.Put("currentPath", i.CurrentPath)
	return b.String()
}

var (
	_paneCurrentPath = tmuxfmt.Var("pane_current_path")
	_paneID          = tmuxfmt.Var("pane_id")
	_paneWidth       = tmuxfmt.Var("pane_width")
	_paneHeight      = tmuxfmt.Var("pane_height")
	_paneMode        = tmuxfmt.Ternary{
		Cond: tmuxfmt.Var("pane_in_mode"),
		Then: tmuxfmt.Var("pane_mode"),
		Else: tmuxfmt.String("normal-mode"),
	}
	_paneInCopyMode = tmuxfmt.Binary{
		LHS: tmuxfmt.Var("pane_mode"),
		Op:  tmuxfmt.Equals,
		RHS: tmuxfmt.String("copy-mode"),
	}
	_paneScrollPosition = tmuxfmt.Ternary{
		Cond: _paneInCopyMode,
		Then: tmuxfmt.Var("scroll_position"),
		Else: tmuxfmt.Int(0),
	}
	_windowID     = tmuxfmt.Var("window_id")
	_windowZoomed = tmuxfmt.Var("window_zoomed_flag")
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
	fc.IntVar(&info.Width, _paneWidth)
	fc.IntVar(&info.Height, _paneHeight)
	fc.StringVar((*string)(&info.Mode), _paneMode)
	fc.IntVar(&info.ScrollPosition, _paneScrollPosition)
	fc.BoolVar(&info.WindowZoomed, _windowZoomed)
	fc.StringVar(&info.CurrentPath, _paneCurrentPath)

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
