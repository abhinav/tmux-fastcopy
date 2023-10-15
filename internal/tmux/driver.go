package tmux

import "github.com/abhinav/tmux-fastcopy/internal/stringobj"

// Driver is a low-level API to access tmux. This maps directly to tmux
// commands.
type Driver interface {
	// NewSession runs the tmux new-session command and returns its output.
	NewSession(NewSessionRequest) ([]byte, error)

	// DisplayMessage runs the tmux display-message command and returns its
	// output.
	DisplayMessage(DisplayMessageRequest) ([]byte, error)

	// CapturePane runs the tmux capture-pane command and returns its
	// output.
	CapturePane(CapturePaneRequest) ([]byte, error)

	// SwapPane runs the tmux swap-pane command.
	SwapPane(SwapPaneRequest) error

	// ResizePane runs the tmux resize-pane command.
	ResizePane(ResizePaneRequest) error

	// ResizeWindow runs the tmux resize-window command.
	ResizeWindow(ResizeWindowRequest) error

	// WaitForSignal runs the tmux wait-for command, waiting for a
	// corresponding SendSignal command.
	WaitForSignal(string) error

	// SendSignal runs the tmux wait-for command, activating anyone waiting
	// for this signal.
	SendSignal(string) error

	// ShowOptions runs the tmux show-options command and returns its
	// output.
	ShowOptions(ShowOptionsRequest) ([]byte, error)

	// SetOption runs the tmux set-option command.
	SetOption(SetOptionRequest) error
}

// SetOptionRequest specifies the parameters for the set-option command.
type SetOptionRequest struct {
	// Name of the option to set.
	Name string

	// Value to set the option to.
	Value string

	// Whether this option should be changed globally.
	Global bool
}

// NewSessionRequest specifies the parameter for a new-session command.
type NewSessionRequest struct {
	// Name of the session, if any.
	Name string

	// Output format, if any. Without this, NewSession will not return any
	// output.
	Format string

	// Size of the new window.
	Width, Height int

	// Whether the new session should be detached from this client.
	Detached bool

	// Additional environment variables to pass to the command in the new
	// session.
	Env []string

	// Command to run in this new window. Must have at least one element.
	Command []string
}

func (r NewSessionRequest) String() string {
	var b stringobj.Builder
	b.Put("name", r.Name)
	b.Put("format", r.Format)
	b.Put("width", r.Width)
	b.Put("height", r.Height)
	b.Put("detached", r.Detached)
	b.Put("env", r.Env)
	b.Put("command", r.Command)
	return b.String()
}

// CapturePaneRequest specifies the parameters for a capture-pane command.
type CapturePaneRequest struct {
	// Pane to capture. Defaults to current.
	Pane string

	// Start and end positions of the captured text. Negative lines are
	// positions in history.
	StartLine, EndLine int
}

func (r CapturePaneRequest) String() string {
	var b stringobj.Builder
	b.Put("pane", r.Pane)
	b.Put("startLine", r.StartLine)
	b.Put("endLine", r.EndLine)
	return b.String()
}

// DisplayMessageRequest specifies the parameters for a display-message
// command.
type DisplayMessageRequest struct {
	// Pane to capture. Defaults to current.
	Pane string

	// Message to display.
	Message string
}

func (r DisplayMessageRequest) String() string {
	var b stringobj.Builder
	b.Put("pane", r.Pane)
	b.Put("message", r.Message)
	return b.String()
}

// SwapPaneRequest specifies the parameters for a swap-pane command.
type SwapPaneRequest struct {
	// Source pane. Defaults to current.
	Source string

	// Destination pane to swap the source with.
	Destination string
}

func (r SwapPaneRequest) String() string {
	var b stringobj.Builder
	b.Put("source", r.Source)
	b.Put("destination", r.Destination)
	return b.String()
}

// ResizeWindowRequest specifies the parameters for a resize-window command.
type ResizeWindowRequest struct {
	Window        string
	Width, Height int
}

func (r ResizeWindowRequest) String() string {
	var b stringobj.Builder
	b.Put("window", r.Window)
	b.Put("width", r.Width)
	b.Put("height", r.Height)
	return b.String()
}

// ShowOptionsRequest specifies the parameters for a show-options command.
type ShowOptionsRequest struct {
	Global bool // show global options
}

func (r ShowOptionsRequest) String() string {
	var b stringobj.Builder
	b.Put("global", r.Global)
	return b.String()
}

// ResizePaneRequest specifies the parameters for a resize-pane command.
type ResizePaneRequest struct {
	Target     string // target pane
	ToggleZoom bool   // whether to toggle zoom
}

func (r ResizePaneRequest) String() string {
	var b stringobj.Builder
	b.Put("target", r.Target)
	b.Put("toggleZoom", r.ToggleZoom)
	return b.String()
}
