package tmux

import (
	"io"
	"os/exec"
	"strconv"
	"sync"

	"github.com/abhinav/tmux-fastcopy/internal/log"
)

const _defaultTmux = "tmux"

// ShellDriver is a Driver implementation that shells out to tmux to run
// commands.
type ShellDriver struct {
	// Path to the tmux executable. Defaults to "tmux".
	Path string

	// Logger to write to.
	Log *log.Logger

	// Optional OS environment override. Used in tests.
	env  []string
	once sync.Once
}

var _ Driver = (*ShellDriver)(nil)

func (s *ShellDriver) init() {
	s.once.Do(func() {
		if s.Path == "" {
			s.Path = _defaultTmux
		}

		if s.Log == nil {
			s.Log = log.Discard
		}
	})
}

func (s *ShellDriver) cmd(args ...string) *exec.Cmd {
	cmd := exec.Command(s.Path, args...)
	cmd.Env = s.env
	return cmd
}

// errorWriter sets the provided io.Writers to the same log.Writer and returns
// a function to close them.
//
//   cmd := s.cmd("some", "cmd")
//   defer s.errorWriter(&cmd.Stderr)()
func (s *ShellDriver) errorWriter(ws ...*io.Writer) (close func()) {
	writer := &log.Writer{Log: s.Log, Level: log.Error}
	for _, w := range ws {
		*w = writer
	}
	return func() { writer.Close() }
}

// NewSession runs the tmux new-session command.
func (s *ShellDriver) NewSession(req NewSessionRequest) ([]byte, error) {
	s.init()

	args := []string{"new-session"}
	if n := req.Name; len(n) > 0 {
		args = append(args, "-s", n)
	}
	if fmt := req.Format; len(fmt) > 0 {
		args = append(args, "-P", "-F", fmt)
	}
	if w := req.Width; w > 0 {
		args = append(args, "-x", strconv.Itoa(w))
	}
	if h := req.Height; h > 0 {
		args = append(args, "-y", strconv.Itoa(h))
	}
	if req.Detached {
		args = append(args, "-d")
	}
	for _, env := range req.Env {
		args = append(args, "-e", env)
	}

	args = append(args, req.Command...)
	cmd := s.cmd(args...)
	defer s.errorWriter(&cmd.Stderr)()

	s.Log.Debugf("new session: %v", req)
	return cmd.Output()
}

// CapturePane runs the capture-pane command and returns its output.
func (s *ShellDriver) CapturePane(req CapturePaneRequest) ([]byte, error) {
	s.init()

	args := []string{"capture-pane", "-p"}
	if len(req.Pane) > 0 {
		args = append(args, "-t", string(req.Pane))
	}
	if s := req.StartLine; s != 0 {
		args = append(args, "-S", strconv.Itoa(s))
	}
	if e := req.EndLine; e != 0 {
		args = append(args, "-E", strconv.Itoa(e))
	}
	cmd := s.cmd(args...)
	defer s.errorWriter(&cmd.Stderr)()

	s.Log.Debugf("capture pane: %v", req)
	return cmd.Output()
}

// DisplayMessage displays the given message in tmux and returns its output.
func (s *ShellDriver) DisplayMessage(req DisplayMessageRequest) ([]byte, error) {
	s.init()

	args := []string{"display-message", "-p"}
	if len(req.Pane) > 0 {
		args = append(args, "-t", string(req.Pane))
	}
	args = append(args, req.Message)

	cmd := s.cmd(args...)
	defer s.errorWriter(&cmd.Stderr)()

	s.Log.Debugf("display message: %v", req)
	return cmd.Output()
}

// SwapPane runs the swap-pane command.
func (s *ShellDriver) SwapPane(req SwapPaneRequest) error {
	s.init()

	args := []string{"swap-pane", "-t", req.Destination}
	if s := req.Source; len(s) > 0 {
		args = append(args, "-s", s)
	}
	if req.MaintainZoom {
		args = append(args, "-Z")
	}

	cmd := s.cmd(args...)
	defer s.errorWriter(&cmd.Stdout, &cmd.Stderr)()

	s.Log.Debugf("swap pane: %v", req)
	return cmd.Run()
}

// ResizeWindow runs the resize-window command.
func (s *ShellDriver) ResizeWindow(req ResizeWindowRequest) error {
	s.init()

	args := []string{"resize-window"}
	if w := req.Window; len(w) > 0 {
		args = append(args, "-t", w)
	}

	if w := req.Width; w > 0 {
		args = append(args, "-x", strconv.Itoa(w))
	}
	if h := req.Height; h > 0 {
		args = append(args, "-y", strconv.Itoa(h))
	}

	cmd := s.cmd(args...)
	defer s.errorWriter(&cmd.Stdout, &cmd.Stderr)()

	s.Log.Debugf("resize window: %v", req)
	return cmd.Run()
}

// WaitForSignal runs the wait-for command.
func (s *ShellDriver) WaitForSignal(sig string) error {
	s.init()
	cmd := s.cmd("wait-for", sig)
	defer s.errorWriter(&cmd.Stdout, &cmd.Stderr)()

	s.Log.Debugf("wait-for: %v", sig)
	return cmd.Run()
}

// SendSignal runs the wait-for -S command.
func (s *ShellDriver) SendSignal(sig string) error {
	s.init()
	cmd := s.cmd("wait-for", "-S", sig)
	defer s.errorWriter(&cmd.Stdout, &cmd.Stderr)()

	s.Log.Debugf("wait-for -S: %v", sig)
	return cmd.Run()
}

func (s *ShellDriver) SetBuffer(req SetBufferRequest) error {
	s.init()

	args := []string{"set-buffer"}
	if c := req.Client; len(c) > 0 {
		args = append(args, "-t", c)
	}
	args = append(args, req.Data)

	cmd := s.cmd(args...)
	defer s.errorWriter(&cmd.Stdout, &cmd.Stderr)()

	s.Log.Debugf("set buffer: %v", req)
	return cmd.Run()
}
