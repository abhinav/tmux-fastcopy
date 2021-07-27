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
		shell command that handles the selection.
		COMMAND should expect the selection over stdin.
		By default, tmux-fastcopy will update the write to the tmux
		copy buffer.
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
		},
		Log:        logger,
		Tmux:       &tmuxDriver,
		Executable: cmd.Executable,
		Getenv:     cmd.Getenv,
		Getpid:     cmd.Getpid,
	}).Run(cfg)
}
