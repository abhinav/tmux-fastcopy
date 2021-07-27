package ui

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/abhinav/tmux-fastcopy/internal/log"
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
	if app.Root.HandleEvent(ev) {
		return
	}

	switch ev := ev.(type) {
	case *tcell.EventResize:
		app.Screen.Sync()

	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			app.Stop()
		}
	}
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
	pval := recover()
	if pval == nil {
		return
	}

	app.Log.Errorf("panic: %v\n%s", pval, debug.Stack())

	var err error
	switch pval := pval.(type) {
	case string:
		err = errors.New(pval)
	case error:
		err = pval
	default:
		err = fmt.Errorf("panic: %v", pval)
	}

	app.err = err
	app.Stop()
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

	ticker := time.NewTicker(time.Second / time.Duration(fps))
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
