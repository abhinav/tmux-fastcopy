package ui

import (
	"sync"
	"time"

	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/abhinav/tmux-fastcopy/internal/paniclog"
	"github.com/benbjohnson/clock"
	"github.com/gdamore/tcell/v2"
)

const _defaultFPS = 25

// App drives the main UI for the application.
type App struct {
	// Root is the main application widget.
	Root Widget

	// Screen upon which to draw.
	Screen tcell.Screen

	// Logger to post messages to. Optional.
	Log *log.Logger

	// FPS specifies the refresh rate for the UI. Defaults to 25.
	FPS int

	// Clock is the source of time for the app.
	Clock clock.Clock

	once   sync.Once
	err    error // error, if any
	quit   chan struct{}
	events chan tcell.Event
}

func (app *App) init() {
	app.once.Do(func() {
		if app.Log == nil {
			app.Log = log.Discard
		}

		if app.Clock == nil {
			app.Clock = clock.New()
		}

		app.quit = make(chan struct{})
		app.events = make(chan tcell.Event)

		go app.renderLoop()
		go app.streamEvents()
	})
}

// Start starts the app, rendering the root widget on the screen indefinitely
// until Stop is called.
func (app *App) Start() {
	app.init()

	go app.run()
}

// Wait waits until the application is stopped with Stop.
func (app *App) Wait() error {
	<-app.quit
	return app.err
}

func (app *App) run() {
	defer app.handlePanic()

	events := app.events
	for {
		select {
		case <-app.quit:
			return

		case ev, ok := <-events:
			if ok {
				app.handleEvent(ev)
			} else {
				// don't resolve this channel again
				events = nil
			}
		}
	}
}

func (app *App) handleEvent(ev tcell.Event) {
	switch ev := ev.(type) {
	case *tcell.EventResize:
		app.Screen.Sync()
		return

	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			app.Stop()
			return
		}
	}

	app.Root.HandleEvent(ev)
}

// Stop informs the application that it's time to stop. This will cause the Run
// function to unblock and return.
func (app *App) Stop() {
	select {
	case <-app.quit:
		// already closed

	default:
		if app.quit != nil {
			close(app.quit)
		}
	}
}

// Defer this inside goroutines to catch panics inside them.
func (app *App) handlePanic() {
	w := log.Writer{Log: app.Log, Level: log.Error}
	defer w.Close()

	if err := paniclog.Handle(recover(), &w); err != nil {
		app.err = err
		app.Stop()
	}
}

// streams events from tcell to the app.events channel. Blocks until Stop is
// called.
func (app *App) streamEvents() {
	defer app.handlePanic()

	app.Screen.ChannelEvents(app.events, app.quit)
}

// Renders the root widget at the specified FPS.
func (app *App) renderLoop() {
	defer app.handlePanic()

	fps := app.FPS
	if fps == 0 {
		fps = _defaultFPS
	}

	ticker := app.Clock.Ticker(time.Second / time.Duration(fps))
	defer ticker.Stop()

	for {
		select {
		case <-app.quit:
			return

		case <-ticker.C:
			app.Screen.Clear()
			app.Root.Draw(app.Screen)
			app.Screen.Show()
		}
	}
}
