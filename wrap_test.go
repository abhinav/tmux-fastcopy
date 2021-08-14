package main

import (
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/log/logtest"
	"github.com/abhinav/tmux-fastcopy/internal/tdt"
	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/maxatome/go-testdeep/td"
)

type runFunc func(*config) error

func (f runFunc) Run(c *config) error { return f(c) }

func TestWrapperWrapping(_t *testing.T) {
	_t.Parallel()
	ctrl := gomock.NewController(_t)
	t := td.NewT(_t)

	// Our anchors are declared with the outer td.T and used in the inner
	// td.T so persistence is useful.
	t.SetAnchorsPersist(true)

	tests := []struct {
		desc       string
		giveConfig config

		paneInfo tmux.PaneInfo
		options  []string // options reported by tmux show-options

		wantConfig config
	}{
		{
			desc: "minimal",
			paneInfo: tmux.PaneInfo{
				ID:     "%1",
				Width:  80,
				Height: 40,
			},
			wantConfig: config{
				Pane:    "%1",
				LogFile: t.A(td.Ignore(), "").(string),
			},
		},
		{
			desc: "override log",
			giveConfig: config{
				LogFile: "log.txt",
			},
			paneInfo: tmux.PaneInfo{
				ID:     "%2",
				Width:  80,
				Height: 40,
			},
			wantConfig: config{
				Pane:    "%2",
				LogFile: t.A(td.Not("log.txt"), "").(string),
			},
		},
		{
			desc: "has options",
			options: []string{
				"@fastcopy-action pbcopy",
				"@fastcopy-alphabet asdfghjkl",
			},
			paneInfo: tmux.PaneInfo{
				ID:     "%3",
				Width:  80,
				Height: 40,
			},
			wantConfig: config{
				Pane:     "%3",
				Action:   "pbcopy",
				Alphabet: alphabet("asdfghjkl"),
				LogFile:  t.A(td.Not("log.txt"), "").(string),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *td.T) {
			tdt.Parallel(t)

			mockTmux := tmuxtest.NewMockDriver(ctrl)
			mockTmux.EXPECT().NewSession(gomock.Any()).
				Do(func(req tmux.NewSessionRequest) {
					fset := flag.NewFlagSet(_name, flag.ContinueOnError)

					var gotConfig config
					gotConfig.RegisterFlags(fset)

					t.CmpNoError(fset.Parse(req.Command[1:]))
					t.Cmp(gotConfig, tt.wantConfig)
				})

			mockTmux.EXPECT().WaitForSignal(gomock.Any())
			mockTmux.EXPECT().ShowOptions(gomock.Any()).
				Return([]byte(strings.Join(tt.options, "\n")+"\n"), nil)

			w := wrapper{
				Tmux: mockTmux,
				Log:  logtest.NewLogger(t),
				Executable: func() (string, error) {
					return _name, nil
				},
				Getenv: func(string) string { return "" },
				Getpid: func() int { return 42 },
				inspectPane: func(tmux.Driver, string) (*tmux.PaneInfo, error) {
					return &tt.paneInfo, nil
				},
			}
			td.CmpNoError(t, w.Run(&tt.giveConfig))
		})
	}
}

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
