package main

import (
	"errors"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/log/logtest"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_Run_badRegex(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	err := (&app{
		Log:  logtest.NewLogger(t),
		Tmux: tmuxtest.NewMockDriver(mockCtrl),
	}).Run(&config{
		Regexes: regexes{
			"foo": "not(a{valid[regex",
		},
	})
	require.Error(t, err, "run must fail")
	assert.ErrorContains(t, err, `compile regex "foo"`)
}

func TestApp_Run_inspectPaneError(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)

	tmuxDriver := tmuxtest.NewMockDriver(mockCtrl)
	tmuxDriver.EXPECT().
		DisplayMessage(tmuxtest.DisplayMessageRequestMatcher{Pane: "42"}).
		Return(nil, errors.New("great sadness"))

	err := (&app{
		Log:  logtest.NewLogger(t),
		Tmux: tmuxDriver,
	}).Run(&config{Pane: "42"})
	require.Error(t, err, "run must fail")
	assert.ErrorContains(t, err, "great sadness")
}
