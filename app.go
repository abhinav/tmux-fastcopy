package main

import (
	"fmt"
	"regexp"

	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/ui"
	tcell "github.com/gdamore/tcell/v2"
)

var _regex = []*regexp.Regexp{
	regexp.MustCompile(`\d{1,3}(?:\.\d{1,3}){3}`),         // IP v4 addresses
	regexp.MustCompile(`[0-9a-f]{7,40}`),                  // git SHAs
	regexp.MustCompile(`(?i)0x[0-9a-f]{2,}`),              // hex addresses
	regexp.MustCompile(`(?i)#[0-9a-f]{6}`),                // hex colors
	regexp.MustCompile(`-?\d{4,}`),                        // numbers
	regexp.MustCompile(`(?:[\w\-\.]+|~)?(?:/[\w\-\.]+)+`), // paths

	// UUIDs
	regexp.MustCompile(`(?i)[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}`),
}

// app implements the main fastcopy application logic. It assumes that it's
// running inside a tmux window that it has full control over. (wrapper takes
// care of ensuring that.)
type app struct {
	Log       *log.Logger
	Tmux      tmux.Driver
	NewAction func(string) (action, error)

	NewScreen func() (tcell.Screen, error) // == tcell.NewScreen
}

// Run runs the application with the provided configuration.
func (app *app) Run(cfg *config) error {
	cfg.FillFrom(&_defaultConfig)

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
	}
	ctrl.Init()

	if err := app.Tmux.SwapPane(tmux.SwapPaneRequest{
		Source:       targetPane.ID,
		Destination:  myPane.ID,
		MaintainZoom: true,
	}); err != nil {
		return err
	}
	defer func() {
		app.Tmux.SwapPane(tmux.SwapPaneRequest{
			Destination:  targetPane.ID,
			Source:       myPane.ID,
			MaintainZoom: true,
		})
	}()

	selection, err := ctrl.Wait()
	if err != nil {
		return err
	}

	action, err := app.NewAction(cfg.Action)
	if err != nil {
		return fmt.Errorf("load action %q: %v", cfg.Action, err)
	}

	return action.Run(selection)
}

type ctrl struct {
	Screen   tcell.Screen
	Log      *log.Logger
	Alphabet []rune
	Text     string

	w   *Widget
	ui  *ui.App
	sel string
}

func (c *ctrl) Init() {
	var matches [][]int
	for _, re := range _regex {
		matches = append(matches, re.FindAllStringIndex(c.Text, -1)...)
	}
	ms := make([]Range, len(matches))
	for i, m := range matches {
		ms[i] = Range{Start: m[0], End: m[1]}
	}

	base := tcell.StyleDefault.
		Background(tcell.ColorBlack).
		Foreground(tcell.ColorWhite)

	c.w = New(Config{
		Text:         c.Text,
		Matches:      ms,
		Handler:      c,
		HintAlphabet: c.Alphabet,
		Style: Style{
			Normal:         base,
			Match:          base.Foreground(tcell.ColorGreen),
			SkippedMatch:   base.Foreground(tcell.ColorGray),
			HintLabel:      base.Foreground(tcell.ColorRed),
			HintLabelInput: base.Foreground(tcell.ColorYellow),
		},
	})

	c.ui = &ui.App{
		Root:   c.w,
		Screen: c.Screen,
		Log:    c.Log,
	}

	c.ui.Start()
}

func (c *ctrl) Wait() (string, error) {
	err := c.ui.Wait()
	return c.sel, err
}

func (c *ctrl) HandleSelection(_ string, text string) {
	c.sel = text
	c.ui.Stop()
}
