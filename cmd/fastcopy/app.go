package main

import (
	"fmt"
	"io"
	"os"

	"github.com/abhinav/tmux-fastcopy/internal/fastcopy"
	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/abhinav/tmux-fastcopy/internal/ui"
	tcell "github.com/gdamore/tcell/v2"
)

// app implements the main fastcopy application logic. It assumes that it's
// running inside a tmux window that it has full control over. (wrapper takes
// care of ensuring that.)
type app struct {
	Log       *log.Logger
	NewAction func(string) (action, error)

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

	bs, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %v", err)
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

	action, err := app.NewAction(actionStr)
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
