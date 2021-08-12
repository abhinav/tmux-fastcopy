package main

import (
	"errors"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/log/logtest"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/maxatome/go-testdeep/td"
)

type runFunc func(*config) error

func (f runFunc) Run(c *config) error { return f(c) }

func TestWrapperWrapped(t *testing.T) {
	t.Parallel()

	runWrapper := func(t *testing.T, wrapped runFunc) error {
		ctrl := gomock.NewController(t)
		mockTmux := tmuxtest.NewMockDriver(ctrl)
		mockTmux.EXPECT().SendSignal("TMUX_FASTCOPY_WRAPPER_42")

		getenv := func(k string) string {
			if td.Cmp(t, k, _parentPIDEnv, "unexpected var") {
				return "42"
			}
			return ""
		}

		w := wrapper{
			Wrapped: runFunc(wrapped),
			Tmux:    mockTmux,
			Log:     logtest.NewLogger(t),
			Getenv:  getenv,
		}
		return w.Run(nil)
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		ran := false
		defer func() {
			td.CmpTrue(t, ran, "wrapped function never run")
		}()
		wrapped := func(*config) error {
			td.CmpFalse(t, ran, "wrapped function ran twice")
			ran = true
			return nil
		}

		err := runWrapper(t, wrapped)
		td.CmpNoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		wrapped := func(*config) error {
			return errors.New("great sadness")
		}

		err := runWrapper(t, wrapped)
		if td.CmpNotNil(t, err) {
			td.CmpContains(t, err.Error(), "great sadness")
		}
	})

	t.Run("panics", func(t *testing.T) {
		t.Parallel()

		wrapped := func(*config) error {
			panic("great sadness")
		}

		td.CmpPanic(t, func() {
			runWrapper(t, wrapped)
		}, "great sadness")
	})
}
