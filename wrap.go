package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/abhinav/tmux-fastcopy/internal/tail"
	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxopt"
	"go.uber.org/multierr"
)

const (
	_parentPIDEnv = "TMUX_FASTCOPY_WRAPPED_BY"
	_signalPrefix = "TMUX_FASTCOPY_WRAPPER_"
)

// wrapper wraps another function to ensure that it runs in its own tmux
// session that it has full ownership of.
type wrapper struct {
	Wrapped interface{ Run(*config) error } // wrapped command
	Tmux    tmux.Driver
	Log     *log.Logger

	Executable func() (string, error) // os.Executable
	Getenv     func(string) string    // os.Getenv
	Getpid     func() int             // os.Getpid

	// To override tmux.InspectPane for tests.
	inspectPane func(tmux.Driver, string) (*tmux.PaneInfo, error)
}

// Run runs the wrapper with the provided configuration. If we're already
// wrapped in a tmux session, Run calls the wrapped command. Otherwise, it
// calls re-runs the binary in a new tmux session and waits for it to exit.
// Logs written by the wrapped command will be reproduced to the logs for
// wrapper.
func (w *wrapper) Run(cfg *config) (err error) {
	// We work by setting the TMUX_FASTCOPY_WRAPPED_BY environment variable
	// to the PID of the wrapper process. If TMUX_FASTCOPY_WRAPPED_BY is
	// set, we know we're inside the wrapped binary.
	//
	// Further, we use the PID as part of the signal we sent to block and
	// unblock the binary with tmux using the tmux wait-for command, so if
	// the PID is 42, the signal is TMUX_FASTCOPY_WRAPPER_42.

	parent := w.Getenv(_parentPIDEnv)
	if len(parent) > 0 {
		// We're in the wrapped process. Let it run as usual and send a
		// signal to unblock the parent when done.
		defer w.Tmux.SendSignal(_signalPrefix + parent)
		return w.Wrapped.Run(cfg)
	}
	parent = strconv.Itoa(w.Getpid())

	exe, err := w.Executable()
	if err != nil {
		return fmt.Errorf("determine executable: %v", err)
	}

	// Disambiguate the pane identifier to a pane ID. This is unqiue across
	// sessions.
	inspectPane := tmux.InspectPane
	if w.inspectPane != nil {
		inspectPane = w.inspectPane
	}
	pane, err := inspectPane(w.Tmux, cfg.Pane)
	if err != nil {
		return fmt.Errorf("inspect pane %q: %v", cfg.Pane, err)
	}
	cfg.Pane = pane.ID

	// Send the logs to a temporary file that we will copy from until we
	// exit.
	tmpLog, err := ioutil.TempFile("", "tmux-fastcopy")
	if err != nil {
		return err
	}
	defer func() {
		err = multierr.Append(err, os.Remove(tmpLog.Name()))
	}()
	cfg.LogFile = tmpLog.Name()

	tmuxLoader := tmuxopt.Loader{Tmux: w.Tmux}
	var tmuxCfg config
	tmuxCfg.RegisterOptions(&tmuxLoader)
	if err := tmuxLoader.Load(tmux.ShowOptionsRequest{Global: true}); err != nil {
		return fmt.Errorf("load options: %v", err)
	}

	cfg.FillFrom(&tmuxCfg)

	req := tmux.NewSessionRequest{
		Width:    pane.Width,
		Height:   pane.Height,
		Detached: true,
		Env:      []string{fmt.Sprintf("%v=%v", _parentPIDEnv, parent)},
		Command:  append([]string{exe}, cfg.Flags()...),
	}
	if _, err := w.Tmux.NewSession(req); err != nil {
		return err
	}

	logw := &log.Writer{Log: w.Log}
	defer logw.Close()

	tee := tail.Tee{W: logw, R: tmpLog}
	tee.Start()
	defer func() {
		if terr := tee.Stop(); terr != nil {
			err = multierr.Append(err, fmt.Errorf("stopped copying logs: %v", terr))
		}
	}()

	return w.Tmux.WaitForSignal(_signalPrefix + parent)
}
