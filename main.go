// tmux-fastcopy is a plugin for tmux that aids in copying text.
// It allows matching text on the screen with pre-defined regular expressions
// and copying the matched text with minimal keystrokes.
//
// See README.md for more information.
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

var _main = mainCmd{
	Stdout:     os.Stdout,
	Stderr:     os.Stderr,
	Executable: os.Executable,
	Getenv:     os.Getenv,
	Environ:    os.Environ,
	Getpid:     os.Getpid,
}

func main() {
	if err := run(&_main, os.Args[1:]); err != nil && err != flag.ErrHelp {
		fmt.Fprintln(_main.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *mainCmd, args []string) (err error) {
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
		fmt.Fprintln(cmd.Stdout, "tmux-fastcopy version", _version)
		fmt.Fprintln(cmd.Stdout, "Copyright (C) 2023 Abhinav Gupta")
		fmt.Fprintln(cmd.Stdout, "  <https://github.com/abhinav/tmux-fastcopy>")
		fmt.Fprintln(cmd.Stdout, "tmux-fastcopy comes with ABSOLUTELY NO WARRANTY.")
		fmt.Fprintln(cmd.Stdout, "This is free software, and you are welcome to redistribute it")
		fmt.Fprintln(cmd.Stdout, "under certain conditions. See source for details.")
		return nil
	}

	if args := flag.Args(); len(args) > 0 {
		return fmt.Errorf("unexpected arguments %q", args)
	}

	return cmd.Run(&cfg)
}

type mainCmd struct {
	Stdout io.Writer
	Stderr io.Writer

	Executable func() (string, error) // == os.Executable
	Getenv     func(string) string    // == os.Getenv
	Environ    func() []string        // == os.Environ
	Getpid     func() int

	newTmuxDriver func(string) tmuxShellDriver
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
	-shift-action COMMAND
		command and arguments that handle the selection.
		'action' specifies the default selection action, and
		'shift-action' specifies the action with the Shift key pressed.
		The first '{}' in the argument list is the selected text.
		If there is no '{}', the selected text is sent over stdin.
			-action 'tmux load-buffer -'  # default
			-action pbcopy -shift-action open
		Uses 'tmux load-buffer' by default for 'action' and no-op for
		'shift-action'.
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
		Actions receive the name of the matching regex in the
		FASTCOPY_REGEX_NAME environment variable.
		Default set includes: ipv4, gitsha, hexaddr, hexcolor, int,
		path, uuid.
	-alphabet STRING
		characters used to generate labels.
			-alphabet "asdfghjkl;"  # qwerty home row
		Uses the English alphabet by default.
	-tmux PATH
		path to tmux executable.
			-tmux /usr/bin/tmux
		Searches $PATH for tmux by default.
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
		cmd.newTmuxDriver = func(path string) tmuxShellDriver {
			return &tmux.ShellDriver{Path: path}
		}
	}

	if cmd.runTarget == nil {
		cmd.runTarget = runTarget
	}
}

func (cmd *mainCmd) Run(cfg *config) (err error) {
	cmd.init()

	if file := cfg.LogFile; len(file) > 0 {
		f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open log %q: %v", file, err)
		}
		defer multierr.AppendInvoke(&err, multierr.Close(f))
		cmd.Stderr = f
	}

	tmuxDriver := cmd.newTmuxDriver(cfg.Tmux)

	// If we're wrapped, wait to send the done signal *after* writing the
	// panic.
	parent := cmd.Getenv(_parentPIDEnv)
	if len(parent) > 0 {
		defer func(signal string) {
			err = multierr.Append(err, tmuxDriver.SendSignal(signal))
		}(_signalPrefix + parent)
	}
	defer paniclog.Recover(&err, cmd.Stderr)

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
				Log:     logger,
				Environ: cmd.Environ,
				Getwd:   os.Getwd,
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

	return cmd.runTarget(target, cfg)
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
