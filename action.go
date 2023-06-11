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
)

func regexNamesEnvEntry(matchers []string) string {
	return _regexNamesEnvKey + "=" + strings.Join(matchers, " ")
}

type actionFactory struct {
	Log     *log.Logger
	Environ func() []string
}

// New builds a command handler from the provided string.
//
// The string is a multi-word shell command. It should use "{}" as an argument
// to reference the selected text. If no "{}" is present, the selection will be
// sent to the command over stdin.
func (f *actionFactory) New(action string) (action, error) {
	args, err := shellwords.Parse(action)
	if err != nil {
		return nil, err
	}

	if len(args) == 0 {
		return nil, errors.New("empty action")
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
			}, nil
		}
	}

	// No "{}" use stdin.
	return &stdinAction{
		Cmd:     cmd,
		Args:    args,
		Log:     f.Log,
		Environ: f.Environ,
	}, nil
}

// action specifies how to handle the user's selection.
type action interface {
	Run(fastcopy.Selection) error
}

type stdinAction struct {
	Cmd     string
	Args    []string
	Log     *log.Logger
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
	cmd.Env = append(h.Environ(), regexNamesEnvEntry(sel.Matchers))
	return cmd.Run()
}

type argAction struct {
	Cmd                   string
	BeforeArgs, AfterArgs []string
	Log                   *log.Logger
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
	cmd.Env = append(h.Environ(), regexNamesEnvEntry(sel.Matchers))
	return cmd.Run()
}
