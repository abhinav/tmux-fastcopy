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

var _version = "dev"

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

const _name = "tmux-fastcopy"

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
		characters used to generate labels.
			-alphabet "asdfghjkl;"  # qwerty home row
		Uses the English alphabet by default.
	-log FILE
		file to write logs to.
		Uses stderr by default.
	-verbose
		log more output.
	-version
		display version information.
`

func (cmd *mainCmd) Run(args []string) error {
	var cfg config

	flag := flag.NewFlagSet(_name, flag.ContinueOnError)
	flag.SetOutput(cmd.Stderr)
	flag.Usage = func() {
		name := flag.Name()
		fmt.Fprintf(flag.Output(), _usage, name)
	}
	cfg.RegisterFlags(flag)
	version := flag.Bool("version", false, "")
	if err := flag.Parse(args); err != nil {
		return err
	}

	if *version {
		fmt.Fprintf(cmd.Stdout, "tmux-fastcopy version %v\n", _version)
		return nil
	}

	if args := flag.Args(); len(args) > 0 {
		return fmt.Errorf("unexpected arguments %q", args)
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

	tmuxDriver := tmux.ShellDriver{Log: logger.WithName("tmux")}

	return (&wrapper{
		Wrapped: &app{
			Log:       logger,
			Tmux:      &tmuxDriver,
			NewScreen: tcell.NewScreen,
			NewAction: (&actionFactory{
				Log: logger,
			}).New,
		},
		Log:        logger,
		Tmux:       &tmuxDriver,
		Executable: cmd.Executable,
		Getenv:     cmd.Getenv,
		Getpid:     cmd.Getpid,
	}).Run(&cfg)
}
