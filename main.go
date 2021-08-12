package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	tcell "github.com/gdamore/tcell/v2"
)

func main() {
	cmd := mainCmd{
		Stdin:      os.Stdin,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Executable: os.Executable,
		Getenv:     os.Getenv,
		Getpid:     os.Getpid,
	}
	if err := cmd.Run(os.Args[1:]); err != nil && err != flag.ErrHelp {
		fmt.Fprintln(cmd.Stderr, err)
		os.Exit(1)
	}
}

type mainCmd struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	Executable func() (string, error) // == os.Executable
	Getenv     func(string) string    // == os.Getenv
	Getpid     func() int
}

const _usage = `usage: %v [options]

Renders a vimium/vimperator-style overlay on top of the text in a tmux window
to allow copying important text on the screen.

The following flags are available:

	-pane PANE
		target pane for the overlay.
		This may be a pane index in the current window, or a unique
		pane identifier.
		Uses the current pane if unspecified.
	-action COMMAND
		command and arguments that handle the selection.
		The first '{}' in the argument list is the selected text.
			-action 'tmux set-buffer -- {}'  # default
		If there is no '{}', the selected text is sent over stdin.
			-action pbcopy
		Uses 'tmux set-buffer' by default.
	-alphabet STRING
		characters used for hints in-order.
			-alphabet "asdfghjkl;"  # qwerty home row
		Uses the English alphabet by default.
	-log FILE
		file to write logs to.
		Uses stderr by default.
	-verbose
		log more output.
`

func (cmd *mainCmd) Run(args []string) error {
	flag := flag.NewFlagSet("tmux-fastcopy", flag.ContinueOnError)
	flag.SetOutput(cmd.Stderr)
	flag.Usage = func() {
		name := flag.Name()
		fmt.Fprintf(flag.Output(), _usage, name)
	}

	cfg := newConfig(flag)

	if err := flag.Parse(args); err != nil {
		return err
	}

	if args := flag.Args(); len(args) > 0 {
		return fmt.Errorf("unexpected arguments %q", args)
	}

	if len(cfg.Action) == 0 {
		cfg.Action = _defaultAction
	}

	if alpha := cfg.Alphabet; len(alpha) > 0 {
		if err := validateAlphabet(alpha); err != nil {
			return err
		}
	} else {
		cfg.Alphabet = _defaultAlphabet
	}

	logW, closeLog, err := cfg.BuildLogWriter(cmd.Stderr)
	if err != nil {
		return err
	}
	defer closeLog()

	logger := log.New(logW)
	if cfg.Verbose {
		logger = logger.WithLevel(log.Debug)
	}

	action, err := (&actionFactory{Log: logger}).New(cfg.Action)
	if err != nil {
		return fmt.Errorf("load action %q: %v", cfg.Action, err)
	}

	tmuxDriver := tmux.ShellDriver{Log: logger.WithName("tmux")}

	return (&wrapper{
		Wrapped: &app{
			Log:       logger,
			Tmux:      &tmuxDriver,
			NewScreen: tcell.NewScreen,
			Action:    action,
		},
		Log:        logger,
		Tmux:       &tmuxDriver,
		Executable: cmd.Executable,
		Getenv:     cmd.Getenv,
		Getpid:     cmd.Getpid,
	}).Run(cfg)
}
