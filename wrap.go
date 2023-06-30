package main

import (
	"bytes"
	"fmt"
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
	Tmux tmux.Driver
	Log  *log.Logger

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
	tmpLog, err := os.CreateTemp("", "tmux-fastcopy")
	if err != nil {
		return err
	}
	defer func() {
		err = multierr.Append(err, os.Remove(tmpLog.Name()))
	}()

	tmuxLoader := tmuxopt.Loader{Tmux: w.Tmux}
	var tmuxCfg config
	tmuxCfg.RegisterOptions(&tmuxLoader)
	if err := tmuxLoader.Load(tmux.ShowOptionsRequest{Global: true}); err != nil {
		return fmt.Errorf("load options: %v", err)
	}

	cfg.LogFile = tmpLog.Name()
	cfg.FillFrom(&tmuxCfg)

	targetPane, err := tmux.InspectPane(w.Tmux, "")
	if err != nil {
		return fmt.Errorf("inspect pane %q: %v", cfg.Pane, err)
	}

	creq := tmux.CapturePaneRequest{Pane: targetPane.ID}
	if targetPane.Mode == tmux.CopyMode {
		// If the pane is in copy-mode, the default capture-pane will
		// capture the bottom of the screen that would normally be
		// visible if not in copy mode. Supply positions to capture for
		// that case.
		creq.StartLine = -targetPane.ScrollPosition
		creq.EndLine = creq.StartLine + targetPane.Height - 1
	}

	bs, err := w.Tmux.CapturePane(creq)
	if err != nil {
		return fmt.Errorf("capture pane %q: %v", cfg.Pane, err)
	}

	// TODO: somehow send this to NewSession
	_ = bs
	// TODO tmux pipe-pane?
	// TODO send-keys -H 04 to send EOT

	parent := strconv.Itoa(w.Getpid())
	sessionID, err := w.Tmux.NewSession(tmux.NewSessionRequest{
		Width:    pane.Width,
		Height:   pane.Height,
		Detached: true,
		Env: []string{
			fmt.Sprintf("%v=%v", _parentPIDEnv, parent),
		},
		Command: append([]string{exe}, cfg.Flags()...),
		Format:  "#{session_id}",
	})
	if err != nil {
		return fmt.Errorf("start tmux session: %w", err)
	}

	panes, err := w.Tmux.ListPanes(tmux.ListPanesRequest{
		Session: string(sessionID),
	})
	if err != nil {
		return fmt.Errorf("list panes: %w", err)
	}
	if len(panes) != 1 {
		panesJoined := bytes.Join(panes, []byte(", "))
		return fmt.Errorf("expected 1 pane, got %v: %s", len(panes), panesJoined)
	}

	// Size specification in new-session doesn't always take and causes
	// flickers when swapping panes around. Make sure that the window is
	// right-sized.
	fastcopyPane, err := tmux.InspectPane(w.Tmux, string(panes[0]))
	if err != nil {
		return err
	}

	if fastcopyPane.Width != targetPane.Width || fastcopyPane.Height != targetPane.Height {
		resizeReq := tmux.ResizeWindowRequest{
			Window: fastcopyPane.WindowID,
			Width:  targetPane.Width,
			Height: targetPane.Height,
		}
		if err := w.Tmux.ResizeWindow(resizeReq); err != nil {
			w.Log.Errorf("unable to resize %q: %v", fastcopyPane.WindowID, err)
			// Not the end of the world. Keep going.
		}
	}

	logw := &log.Writer{Log: w.Log}
	defer multierr.AppendInvoke(&err, multierr.Close(logw))

	tee := tail.Tee{W: logw, R: tmpLog}
	tee.Start()
	defer func() {
		err = multierr.Append(err, tmpLog.Close())
		err = multierr.Append(err, tee.Stop())
	}()

	if err := w.Tmux.SwapPane(tmux.SwapPaneRequest{
		Source:      targetPane.ID,
		Destination: fastcopyPane.ID,
	}); err != nil {
		return err
	}

	// If the window was zoomed, zoom the swapped pane as well. In Tmux 3.1
	// or newer, we can use the '-Z' flag of swap-pane, but that's not
	// available in older versions.
	if targetPane.WindowZoomed {
		_ = w.Tmux.ResizePane(tmux.ResizePaneRequest{
			Target:     fastcopyPane.ID,
			ToggleZoom: true,
		})

		defer func() {
			_ = w.Tmux.ResizePane(tmux.ResizePaneRequest{
				Target:     targetPane.ID,
				ToggleZoom: true,
			})
		}()
	}

	defer func() {
		_ = w.Tmux.SwapPane(tmux.SwapPaneRequest{
			Destination: targetPane.ID,
			Source:      fastcopyPane.ID,
		})
	}()

	return w.Tmux.WaitForSignal(_signalPrefix + parent)
}
