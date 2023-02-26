package tmux

import (
	"errors"
	"io"
	"os/exec"
	"strconv"
	"sync"

	"github.com/abhinav/tmux-fastcopy/internal/log"
)

const (
	_defaultTmux = "tmux"
	_defaultEnv  = "/usr/bin/env"
)

// minimal hook to change how exec.Cmd are run. Tests will provide a different
// implementation.
type runner struct {
	Run    func(*exec.Cmd) error
	Output func(*exec.Cmd) ([]byte, error)
}

var defaultRunner = runner{
	Run:    (*exec.Cmd).Run,
	Output: (*exec.Cmd).Output,
}

// ShellDriver is a Driver implementation that shells out to tmux to run
// commands.
type ShellDriver struct {
	// Path to the tmux executable. Defaults to "tmux".
	Path string

	// Path to the env command. Defaults to /usr/bin/env.
	Env string

	log  *log.Logger
	run  *runner
	once sync.Once
}

var _ Driver = (*ShellDriver)(nil)

func (s *ShellDriver) init() {
	s.once.Do(func() {
		if s.log == nil {
			s.log = log.Discard
		}

		if s.Path == "" {
			s.Path = _defaultTmux
		}

		if s.Env == "" {
			s.Env = _defaultEnv
		}

		if s.run == nil {
			s.run = &defaultRunner
		}
	})
}

// SetLogger specifies the logger for the ShellDriver. By default, the
// ShellDriver does not log anything.
func (s *ShellDriver) SetLogger(log *log.Logger) {
	s.log = log
}

func (s *ShellDriver) cmd(args ...string) *exec.Cmd {
	cmd := exec.Command(s.Path, args...)
	return cmd
}

// errorWriter sets the provided io.Writers to the same log.Writer and returns
// a function to close them.
//
//	cmd := s.cmd("some", "cmd")
//	defer s.errorWriter(&cmd.Stderr)()
func (s *ShellDriver) errorWriter(ws ...*io.Writer) (close func()) {
	writer := &log.Writer{Log: s.log, Level: log.Error}
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

	// We could use the -e flg to set the environment variables, but that
	// was added in tmux 3.2. Instead, use,
	//
	//   /usr/bin/env K1=V1 K2=V2 cmd "$1" "$2" ...
	if len(req.Env) > 0 {
		if len(req.Command) == 0 {
			return nil, errors.New("env can be set only if command is set")
		}
		setenv := make([]string, len(req.Env)+1)
		setenv[0] = s.Env
		copy(setenv[1:], req.Env)
		args = append(args, setenv...)
	}

	args = append(args, req.Command...)
	cmd := s.cmd(args...)
	defer s.errorWriter(&cmd.Stderr)()

	s.log.Debugf("new session: %v", req)
	return s.run.Output(cmd)
}

// CapturePane runs the capture-pane command and returns its output.
func (s *ShellDriver) CapturePane(req CapturePaneRequest) ([]byte, error) {
	s.init()

	args := []string{"capture-pane", "-p", "-J"}
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

	s.log.Debugf("capture pane: %v", req)
	return s.run.Output(cmd)
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

	s.log.Debugf("display message: %v", req)
	return s.run.Output(cmd)
}

// SwapPane runs the swap-pane command.
func (s *ShellDriver) SwapPane(req SwapPaneRequest) error {
	s.init()

	args := []string{"swap-pane", "-t", req.Destination}
	if s := req.Source; len(s) > 0 {
		args = append(args, "-s", s)
	}

	cmd := s.cmd(args...)
	defer s.errorWriter(&cmd.Stdout, &cmd.Stderr)()

	s.log.Debugf("swap pane: %v", req)
	return s.run.Run(cmd)
}

// ResizePane runs the resize-pane command.
func (s *ShellDriver) ResizePane(req ResizePaneRequest) error {
	s.init()

	args := []string{"resize-pane", "-t", req.Target}
	if req.ToggleZoom {
		args = append(args, "-Z")
	}

	cmd := s.cmd(args...)
	defer s.errorWriter(&cmd.Stdout, &cmd.Stderr)()

	s.log.Debugf("resize pane: %v", req)
	return s.run.Run(cmd)
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

	s.log.Debugf("resize window: %v", req)
	return s.run.Run(cmd)
}

// WaitForSignal runs the wait-for command.
func (s *ShellDriver) WaitForSignal(sig string) error {
	s.init()
	cmd := s.cmd("wait-for", sig)
	defer s.errorWriter(&cmd.Stdout, &cmd.Stderr)()

	s.log.Debugf("wait-for: %v", sig)
	return s.run.Run(cmd)
}

// SendSignal runs the wait-for -S command.
func (s *ShellDriver) SendSignal(sig string) error {
	s.init()
	cmd := s.cmd("wait-for", "-S", sig)
	defer s.errorWriter(&cmd.Stdout, &cmd.Stderr)()

	s.log.Debugf("wait-for -S: %v", sig)
	return s.run.Run(cmd)
}

// ShowOptions runs the show-options command.
func (s *ShellDriver) ShowOptions(req ShowOptionsRequest) ([]byte, error) {
	s.init()

	args := []string{"show-options"}
	if req.Global {
		args = append(args, "-g")
	}
	cmd := s.cmd(args...)
	defer s.errorWriter(&cmd.Stderr)()

	s.log.Debugf("show options: %v", req)
	return s.run.Output(cmd)
}
