package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/abhinav/tmux-fastcopy/internal/paniclog"
	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	tcell "github.com/gdamore/tcell/v2"
	"go.uber.org/multierr"
)

var _version = "dev"

func main() {
	cmd := mainCmd{
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
	Stdout io.Writer
	Stderr io.Writer

	Executable func() (string, error) // == os.Executable
	Getenv     func(string) string    // == os.Getenv
	Getpid     func() int

	newTmuxDriver func() tmuxShellDriver
	runTarget     runTargetFunc
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
	-regex NAME:PATTERN
		regular expressions to search for.
		Name identifies the pattern. Add this option any number of
		times.
			-regex 'attr:\w+\.\w+'
		Use prior names to replace or unset patterns.
			-regex 'ipv4:'
		Capture groups in the regex indicate the text to be copied,
		defaulting to the whole string if there are no capture groups.
			-regex 'gitsha:([0-9a-f]{7})[0-9a-f]{,33}'
		Default set includes: ipv4, gitsha, hexaddr, hexcolor, int,
		path, uuid.
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

func (cmd *mainCmd) init() {
	if cmd.newTmuxDriver == nil {
		cmd.newTmuxDriver = func() tmuxShellDriver {
			return new(tmux.ShellDriver)
		}
	}

	if cmd.runTarget == nil {
		cmd.runTarget = runTarget
	}
}

func (cmd *mainCmd) Run(args []string) (err error) {
	cmd.init()

	tmuxDriver := cmd.newTmuxDriver()

	if file := cmd.Getenv(_logfileEnv); len(file) > 0 {
		f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open log %q: %v", file, err)
		}
		defer multierr.AppendInvoke(&err, multierr.Close(f))
		cmd.Stderr = f
	}

	// If we're wrapped, wait to send the done signal *after* writing the
	// panic.
	parent := cmd.Getenv(_parentPIDEnv)
	if len(parent) > 0 {
		defer func(signal string) {
			err = multierr.Append(err, tmuxDriver.SendSignal(signal))
		}(_signalPrefix + parent)
	}

	defer paniclog.Recover(&err, cmd.Stderr)

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

	logger := log.New(cmd.Stderr)
	if cfg.Verbose {
		logger = logger.WithLevel(log.Debug)
	}
	tmuxDriver.SetLogger(logger.WithName("tmux"))

	var target interface{ Run(*config) error }
	if len(parent) > 0 {
		target = &app{
			Log:       logger,
			Tmux:      tmuxDriver,
			NewScreen: tcell.NewScreen,
			NewAction: (&actionFactory{
				Log: logger,
			}).New,
		}
	} else {
		target = &wrapper{
			Log:        logger,
			Tmux:       tmuxDriver,
			Executable: cmd.Executable,
			Getenv:     cmd.Getenv,
			Getpid:     cmd.Getpid,
		}
	}

	return cmd.runTarget(target, &cfg)
}

type tmuxShellDriver interface {
	tmux.Driver

	SetLogger(*log.Logger)
}

// runTargetFunc runs objects that conform to the wrapper/app signatures. This
// type is intentionally cumbersome because it's not meant to be used widely.
type runTargetFunc func(interface {
	Run(*config) error
}, *config) error

func runTarget(target interface{ Run(*config) error }, cfg *config) error {
	return target.Run(cfg)
}
