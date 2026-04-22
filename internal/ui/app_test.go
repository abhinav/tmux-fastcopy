package ui

import (
	"bytes"
	"testing"
	"time"

	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/abhinav/tmux-fastcopy/internal/log/logtest"
	tcell "github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

//nolint:paralleltest // shared state between subtests
func TestAppEvents(t *testing.T) {
	t.Parallel()

	newApp := func(screen tcell.Screen, widget *MockWidget) *App {
		t.Helper()
		return &App{
			Root:   widget,
			Screen: screen,
			Log:    logtest.NewLogger(t),
		}
	}

	t.Run("resize", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		term, scr, fini := NewTestScreen(t, 80, 40)
		widget := NewMockWidget(ctrl)
		widget.EXPECT().Draw(gomock.Any()).AnyTimes()
		app := newApp(scr, widget)
		app.Start()
		defer func() {
			app.Stop()
			assert.NoError(t, app.Wait())
			fini()
		}()

		term.SetSize(vt.Coord{X: 100, Y: 60})
	})

	t.Run("handled action", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		term, scr, fini := NewTestScreen(t, 80, 40)
		widget := NewMockWidget(ctrl)
		widget.EXPECT().Draw(gomock.Any()).AnyTimes()
		app := newApp(scr, widget)
		app.Start()
		defer func() {
			app.Stop()
			assert.NoError(t, app.Wait())
			fini()
		}()

		done := make(chan struct{})
		widget.EXPECT().
			HandleEvent(gomock.Any()).
			DoAndReturn(func(tcell.Event) bool {
				close(done)
				return true
			})

		term.KeyTap(vt.KeyF)
		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("widget did not receive injected key event")
		}
	})

	t.Run("quit/escape", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		term, scr, fini := NewTestScreen(t, 80, 40)
		widget := NewMockWidget(ctrl)
		widget.EXPECT().Draw(gomock.Any()).AnyTimes()
		app := newApp(scr, widget)
		app.Start()
		defer func() {
			app.Stop()
			assert.NoError(t, app.Wait())
			fini()
		}()

		term.KeyTap(vt.KeyEsc)

		// If this deadlocks, esc didn't quit.
		assert.NoError(t, app.Wait())
	})

	t.Run("quit/ctrl-c", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		term, scr, fini := NewTestScreen(t, 80, 40)
		widget := NewMockWidget(ctrl)
		widget.EXPECT().Draw(gomock.Any()).AnyTimes()
		app := newApp(scr, widget)
		app.Start()
		defer func() {
			app.Stop()
			assert.NoError(t, app.Wait())
			fini()
		}()

		term.KeyTap(vt.KeyLCtrl, vt.KeyC)
		assert.NoError(t, app.Wait())
	})
}

func TestAppStopWithoutPendingInput(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	_, scr, fini := NewTestScreen(t, 80, 40)
	defer fini()

	widget := NewMockWidget(ctrl)
	widget.EXPECT().Draw(gomock.Any()).AnyTimes()

	app := App{
		Root:   widget,
		Screen: scr,
		Log:    logtest.NewLogger(t),
	}
	app.Start()

	done := make(chan error, 1)
	go func() {
		done <- app.Wait()
	}()

	app.Stop()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Wait did not unblock after Stop")
	}
}

func TestAppPanic(t *testing.T) {
	t.Parallel()

	assertPanic := func(t *testing.T, app *App, buff *bytes.Buffer) {
		t.Helper()

		err := app.Wait()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "great sadness")
		assert.Contains(t, buff.String(), "panic: great sadness")
		assert.Contains(t, buff.String(), "TestAppPanic")
		assert.Contains(t, buff.String(), "app_test.go")
	}

	t.Run("event panic", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		term, scr, fini := NewTestScreen(t, 80, 40)
		defer fini()

		widget := NewMockWidget(ctrl)
		widget.EXPECT().Draw(gomock.Any()).AnyTimes()

		var buff bytes.Buffer
		app := App{
			Root:   widget,
			Screen: scr,
			Log:    log.New(&buff),
		}
		app.Start()

		widget.EXPECT().
			HandleEvent(gomock.Any()).
			Do(func(tcell.Event) {
				panic("great sadness")
			})

		term.KeyTap(vt.KeyF)
		assertPanic(t, &app, &buff)
	})

	t.Run("render panic", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		_, scr, fini := NewTestScreen(t, 80, 40)
		defer fini()

		widget := NewMockWidget(ctrl)
		widget.EXPECT().Draw(gomock.Any()).
			Do(func(tcell.Screen) {
				panic("great sadness")
			})

		var buff bytes.Buffer
		app := App{
			Root:   widget,
			Screen: scr,
			Log:    log.New(&buff),
		}
		app.Start()

		assertPanic(t, &app, &buff)
	})
}
