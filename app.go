package main

import (
	"fmt"

	"github.com/abhinav/tmux-fastcopy/internal/fastcopy"
	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/ui"
	tcell "github.com/gdamore/tcell/v2"
)

// app implements the main fastcopy application logic. It assumes that it's
// running inside a tmux window that it has full control over. (wrapper takes
// care of ensuring that.)
type app struct {
	Log       *log.Logger
	Tmux      tmux.Driver
	NewAction func(newActionRequest) (action, error)

	NewScreen func() (tcell.Screen, error) // == tcell.NewScreen
}

// Run runs the application with the provided configuration.
func (app *app) Run(cfg *config) error {
	cfg.FillFrom(defaultConfig(cfg))

	matcher := make(matcher, 0, len(cfg.Regexes))
	for name, reg := range cfg.Regexes {
		m, err := compileRegexpMatcher(name, reg)
		if err != nil {
			return fmt.Errorf("compile regex %q: %v", name, reg)
		}
		if m != nil {
			matcher = append(matcher, m)
		}
	}

	targetPane, err := tmux.InspectPane(app.Tmux, cfg.Pane)
	if err != nil {
		return fmt.Errorf("inspect pane %q: %v", cfg.Pane, err)
	}

	// Size specification in new-session doesn't always take and causes
	// flickers when swapping panes around. Make sure that the window is
	// right-sized.
	myPane, err := tmux.InspectPane(app.Tmux, "")
	if err != nil {
		return err
	}

	if myPane.Width != targetPane.Width || myPane.Height != targetPane.Height {
		resizeReq := tmux.ResizeWindowRequest{
			Window: myPane.WindowID,
			Width:  targetPane.Width,
			Height: targetPane.Height,
		}
		if err := app.Tmux.ResizeWindow(resizeReq); err != nil {
			app.Log.Errorf("unable to resize %q: %v",
				myPane.WindowID, err)
			// Not the end of the world. Keep going.
		}
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

	bs, err := app.Tmux.CapturePane(creq)
	if err != nil {
		return fmt.Errorf("capture pane %q: %v", cfg.Pane, err)
	}

	screen, err := app.NewScreen()
	if err != nil {
		return err
	}

	if err := screen.Init(); err != nil {
		return err
	}
	defer screen.Fini()

	ctrl := ctrl{
		Screen:   screen,
		Log:      app.Log,
		Text:     string(bs),
		Alphabet: []rune(cfg.Alphabet),
		Matcher:  matcher,
	}
	ctrl.Init()

	if err := app.Tmux.SwapPane(tmux.SwapPaneRequest{
		Source:      targetPane.ID,
		Destination: myPane.ID,
	}); err != nil {
		return err
	}

	// If the window was zoomed, zoom the swapped pane as well. In Tmux 3.1
	// or newer, we can use the '-Z' flag of swap-pane, but that's not
	// available in older versions.
	if targetPane.WindowZoomed {
		_ = app.Tmux.ResizePane(tmux.ResizePaneRequest{
			Target:     myPane.ID,
			ToggleZoom: true,
		})

		defer func() {
			_ = app.Tmux.ResizePane(tmux.ResizePaneRequest{
				Target:     targetPane.ID,
				ToggleZoom: true,
			})
		}()
	}

	defer func() {
		_ = app.Tmux.SwapPane(tmux.SwapPaneRequest{
			Destination: targetPane.ID,
			Source:      myPane.ID,
		})
	}()

	selection, err := ctrl.Wait()
	if err != nil {
		return err
	}

	actionStr := cfg.Action
	if selection.Shift {
		actionStr = cfg.ShiftAction
	}

	if len(actionStr) == 0 {
		return nil
	}

	action, err := app.NewAction(newActionRequest{
		Action:       actionStr,
		Dir:          targetPane.CurrentPath,
		TargetPaneID: targetPane.ID,
	})
	if err != nil {
		return fmt.Errorf("load action %q: %v", actionStr, err)
	}

	return action.Run(selection)
}

type ctrl struct {
	Screen   tcell.Screen
	Log      *log.Logger
	Alphabet []rune
	Text     string
	Matcher  matcher

	w   *fastcopy.Widget
	ui  *ui.App
	sel fastcopy.Selection
}

func (c *ctrl) Init() {
	base := tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite)

	c.w = (&fastcopy.WidgetConfig{
		Text:         c.Text,
		Matches:      c.Matcher.Match(c.Text),
		Handler:      c,
		HintAlphabet: c.Alphabet,
		Style: fastcopy.Style{
			Normal:         base,
			Match:          base.Foreground(tcell.ColorGreen),
			SkippedMatch:   base.Foreground(tcell.ColorGray),
			HintLabel:      base.Foreground(tcell.ColorRed),
			HintLabelInput: base.Foreground(tcell.ColorYellow),
			SelectedMatch:  base.Foreground(tcell.ColorYellow),
			DeselectLabel:  base.Foreground(tcell.ColorDarkRed),
		},
	}).Build()

	c.ui = &ui.App{
		Root:   c.w,
		Screen: c.Screen,
		Log:    c.Log,
	}

	c.ui.Start()
}

func (c *ctrl) Wait() (fastcopy.Selection, error) {
	err := c.ui.Wait()
	return c.sel, err
}

func (c *ctrl) HandleSelection(sel fastcopy.Selection) {
	c.sel = sel
	c.ui.Stop()
}
