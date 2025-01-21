package main

import (
	"flag"
	"strings"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/envtest"
	"github.com/abhinav/tmux-fastcopy/internal/log/logtest"
	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.abhg.dev/io/ioutil"
)

func TestWrapper(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	tests := []struct {
		desc string

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
				Pane: "%1",
				Tmux: "tmux",
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
				Tmux:     "tmux",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			mockTmux := tmuxtest.NewMockDriver(ctrl)
			mockTmux.EXPECT().NewSession(gomock.Any()).
				Do(func(req tmux.NewSessionRequest) {
					fset := flag.NewFlagSet(_name, flag.ContinueOnError)
					fset.SetOutput(ioutil.TestLogWriter(t, ""))

					var gotConfig config
					gotConfig.RegisterFlags(fset)

					require.NoError(t, fset.Parse(req.Command[1:]))

					// zero out log file for comparison.
					if assert.NotEmpty(t, gotConfig.LogFile, "log file must be specified") {
						gotConfig.LogFile = ""
					}

					assert.Equal(t, tt.wantConfig, gotConfig)
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
				Getenv: envtest.Empty.Getenv,
				Getpid: func() int { return 42 },
				inspectPane: func(tmux.Driver, string) (*tmux.PaneInfo, error) {
					return &tt.paneInfo, nil
				},
			}
			assert.NoError(t, w.Run(&config{}))
		})
	}
}

func TestWrapper_destroyUnattached(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	mockTmux := tmuxtest.NewMockDriver(ctrl)
	mockTmux.EXPECT().ShowOptions(gomock.Any()).
		Return([]byte("destroy-unattached on\n"), nil)
	mockTmux.EXPECT().WaitForSignal(gomock.Any())

	// destroy-unattached is set to off before we create a new session,
	// and set back to on after we create a new session.
	gomock.InOrder(
		mockTmux.EXPECT().SetOption(tmux.SetOptionRequest{
			Global: true,
			Name:   "destroy-unattached",
			Value:  "off",
		}).Return(nil),
		mockTmux.EXPECT().NewSession(gomock.Any()),
		mockTmux.EXPECT().SetOption(tmux.SetOptionRequest{
			Global: true,
			Name:   "destroy-unattached",
			Value:  "on",
		}).Return(nil),
	)

	w := wrapper{
		Tmux: mockTmux,
		Log:  logtest.NewLogger(t),
		Executable: func() (string, error) {
			return _name, nil
		},
		Getenv: envtest.Empty.Getenv,
		Getpid: func() int { return 42 },
		inspectPane: func(tmux.Driver, string) (*tmux.PaneInfo, error) {
			return &tmux.PaneInfo{
				ID:     "%1",
				Width:  80,
				Height: 40,
			}, nil
		},
	}
	assert.NoError(t, w.Run(&config{}))
}
