package main

import (
	"flag"
	"strings"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/envtest"
	"github.com/abhinav/tmux-fastcopy/internal/iotest"
	"github.com/abhinav/tmux-fastcopy/internal/log/logtest"
	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/maxatome/go-testdeep/td"
)

func TestWrapper(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

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
				Pane: "%1",
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
					fset.SetOutput(iotest.Writer(t))

					var gotConfig config
					gotConfig.RegisterFlags(fset)

					td.CmpNoError(t, fset.Parse(req.Command[1:]))
					td.Cmp(t, gotConfig, tt.wantConfig)
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
			td.CmpNoError(t, w.Run(&tt.giveConfig))
		})
	}
}
