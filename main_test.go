package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/envtest"
	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	defer func() {
		assert.Empty(t, stderr.String(), "stderr should be empty")
	}()
	err := (&mainCmd{
		Stdout: &stdout,
		Stderr: &stderr,
		Getenv: envtest.Empty.Getenv,
	}).Run([]string{"-version"})
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), _version)
}

func TestMainLogOverride(t *testing.T) {
	t.Parallel()

	logfile := filepath.Join(t.TempDir(), "log.txt")
	var stdout, stderr bytes.Buffer
	defer func() {
		assert.Empty(t, stdout.String(), "stdout must be empty")
		assert.Empty(t, stderr.String(), "stderr must be empty")
	}()

	err := (&mainCmd{
		Getenv: envtest.MustPairs(_logfileEnv, logfile).Getenv,
		Stdout: &stdout,
		Stderr: &stderr,
	}).Run([]string{"--help"})
	assert.Equal(t, flag.ErrHelp, err)

	body, err := os.ReadFile(logfile)
	require.NoError(t, err)
	assert.Contains(t, string(body), "The following flags are available:")
}

type fakeTmux struct{ tmux.Driver }

func (fakeTmux) SetLogger(*log.Logger) {}

func TestMainParentSignal(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockTmux := tmuxtest.NewMockDriver(ctrl)

	var stdout, stderr bytes.Buffer
	defer func() {
		assert.Empty(t, stdout.String(), "stdout must be empty")
		assert.Empty(t, stderr.String(), "stderr must be empty")
	}()

	mockTmux.EXPECT().
		SendSignal(_signalPrefix + "42").
		Return(nil)

	err := (&mainCmd{
		Stdout: io.Discard,
		Stderr: &stderr,
		Getenv: envtest.MustPairs(_parentPIDEnv, "42").Getenv,
		newTmuxDriver: func() tmuxShellDriver {
			return fakeTmux{mockTmux}
		},
	}).Run([]string{"--version"})
	assert.NoError(t, err)
}

func TestMainTargetPanicWithLog(t *testing.T) {
	t.Parallel()

	logfile := filepath.Join(t.TempDir(), "log.txt")
	var stdout, stderr bytes.Buffer
	defer func() {
		assert.Empty(t, stdout.String(), "stdout must be empty")
		assert.Empty(t, stderr.String(), "stderr must be empty")
	}()

	ctrl := gomock.NewController(t)
	mockTmux := tmuxtest.NewMockDriver(ctrl)

	called := false
	defer func() {
		assert.True(t, called, "runTarget was called")
	}()
	runTarget := func(interface{ Run(*config) error }, *config) error {
		called = true
		panic("great sadness")
	}

	err := (&mainCmd{
		Stdout: &stdout,
		Stderr: &stderr,
		Getenv: envtest.MustPairs(
			_logfileEnv, logfile,
		).Getenv,
		Getpid: func() int { return 42 },
		newTmuxDriver: func() tmuxShellDriver {
			return fakeTmux{mockTmux}
		},
		runTarget: runTarget,
	}).Run(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "great sadness")

	body, err := os.ReadFile(logfile)
	require.NoError(t, err)
	assert.Contains(t, string(body), "panic: great sadness")
}
