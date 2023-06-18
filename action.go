package main

import (
	"errors"
	"os/exec"
	"strings"

	"github.com/abhinav/tmux-fastcopy/internal/fastcopy"
	"github.com/abhinav/tmux-fastcopy/internal/log"
	shellwords "github.com/mattn/go-shellwords"
	"go.uber.org/multierr"
)

const (
	_placeholderArg   = "{}"
	_regexNamesEnvKey = "FASTCOPY_REGEX_NAME"
	_targetPaneEnvKey = "FASTCOPY_TARGET_PANE_ID"
)

func regexNamesEnvEntry(matchers []string) string {
	return _regexNamesEnvKey + "=" + strings.Join(matchers, " ")
}

type actionFactory struct {
	Log     *log.Logger
	Environ func() []string
	Getwd   func() (string, error)
}

type newActionRequest struct {
	// Action is a multi-word shell command.
	// It should use "{}" as an argument to reference the selected text.
	// If no "{}" is present, the selection will be sent to the command
	// over stdin.
	Action string

	// Dir is the working directory to run the command in.
	//
	// If empty, the current working directory is used.
	Dir string

	// TargetPaneID is the ID of the pane to send the output to.
	TargetPaneID string
}

// New builds a command handler from the provided string.
//
// The string is a multi-word shell command. It should use "{}" as an argument
// to reference the selected text. If no "{}" is present, the selection will be
// sent to the command over stdin.
func (f *actionFactory) New(req newActionRequest) (action, error) {
	args, err := shellwords.Parse(req.Action)
	if err != nil {
		return nil, err
	}

	if len(args) == 0 {
		return nil, errors.New("empty action")
	}

	dir := req.Dir
	if dir == "" {
		dir, err = f.Getwd()
		if err != nil {
			// This should never happen, but if it does, use an
			// empty string and let exec.Command() figure it out.
			dir = ""
		}
	}

	cmd, args := args[0], args[1:]
	for i, arg := range args {
		if arg == _placeholderArg {
			return &argAction{
				Cmd:        cmd,
				BeforeArgs: args[:i],
				AfterArgs:  args[i+1:],
				Log:        f.Log,
				Environ:    f.Environ,
				Dir:        dir,
				PaneID:     req.TargetPaneID,
			}, nil
		}
	}

	// No "{}" use stdin.
	return &stdinAction{
		Cmd:     cmd,
		Args:    args,
		Log:     f.Log,
		Environ: f.Environ,
		Dir:     dir,
		PaneID:  req.TargetPaneID,
	}, nil
}

// action specifies how to handle the user's selection.
type action interface {
	Run(fastcopy.Selection) error
}

type stdinAction struct {
	Cmd     string
	Dir     string
	Args    []string
	Log     *log.Logger
	PaneID  string
	Environ func() []string // == os.Environ
}

func (h *stdinAction) Run(sel fastcopy.Selection) (err error) {
	logw := &log.Writer{
		Log: h.Log.WithName(h.Cmd),
	}
	defer multierr.AppendInvoke(&err, multierr.Close(logw))

	cmd := exec.Command(h.Cmd, h.Args...)
	cmd.Stdin = strings.NewReader(sel.Text)
	cmd.Stdout = logw
	cmd.Stderr = logw
	cmd.Dir = h.Dir
	cmd.Env = append(h.Environ(),
		regexNamesEnvEntry(sel.Matchers),
		_targetPaneEnvKey+"="+h.PaneID)
	return cmd.Run()
}

type argAction struct {
	Cmd                   string
	Dir                   string
	BeforeArgs, AfterArgs []string
	Log                   *log.Logger
	PaneID                string
	Environ               func() []string // == os.Environ
}

func (h *argAction) Run(sel fastcopy.Selection) (err error) {
	logw := &log.Writer{
		Log: h.Log.WithName(h.Cmd),
	}
	defer multierr.AppendInvoke(&err, multierr.Close(logw))

	args := make([]string, 0, len(h.BeforeArgs)+len(h.AfterArgs)+1)
	args = append(args, h.BeforeArgs...)
	args = append(args, sel.Text)
	args = append(args, h.AfterArgs...)

	cmd := exec.Command(h.Cmd, args...)
	cmd.Stdout = logw
	cmd.Stderr = logw
	cmd.Dir = h.Dir
	cmd.Env = append(h.Environ(),
		regexNamesEnvEntry(sel.Matchers),
		_targetPaneEnvKey+"="+h.PaneID)
	return cmd.Run()
}
