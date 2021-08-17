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
	"github.com/maxatome/go-testdeep/td"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	defer func() {
		td.CmpEmpty(t, stderr.String(), "stderr should be empty")
	}()
	err := (&mainCmd{
		Stdout: &stdout,
		Stderr: &stderr,
		Getenv: envtest.Empty.Getenv,
	}).Run([]string{"-version"})
	td.CmpNoError(t, err)
	td.CmpContains(t, stdout.String(), _version)
}

func TestMainLogOverride(t *testing.T) {
	t.Parallel()

	logfile := filepath.Join(t.TempDir(), "log.txt")
	var stdout, stderr bytes.Buffer
	defer func() {
		td.CmpEmpty(t, stdout.String(), "stdout must be empty")
		td.CmpEmpty(t, stderr.String(), "stderr must be empty")
	}()

	err := (&mainCmd{
		Getenv: envtest.MustPairs(_logfileEnv, logfile).Getenv,
		Stdout: &stdout,
		Stderr: &stderr,
	}).Run([]string{"--help"})
	td.Cmp(t, err, flag.ErrHelp)

	body, err := os.ReadFile(logfile)
	td.CmpNoError(t, err)
	td.CmpContains(t, string(body), "The following flags are available:")
}

type fakeTmux struct{ tmux.Driver }

func (fakeTmux) SetLogger(*log.Logger) {}

func TestMainParentSignal(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockTmux := tmuxtest.NewMockDriver(ctrl)

	var stdout, stderr bytes.Buffer
	defer func() {
		td.CmpEmpty(t, stdout.String(), "stdout must be empty")
		td.CmpEmpty(t, stderr.String(), "stderr must be empty")
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
	td.CmpNoError(t, err)
}

func TestMainTargetPanicWithLog(t *testing.T) {
	t.Parallel()

	logfile := filepath.Join(t.TempDir(), "log.txt")
	var stdout, stderr bytes.Buffer
	defer func() {
		td.CmpEmpty(t, stdout.String(), "stdout must be empty")
		td.CmpEmpty(t, stderr.String(), "stderr must be empty")
	}()

	ctrl := gomock.NewController(t)
	mockTmux := tmuxtest.NewMockDriver(ctrl)

	called := false
	defer func() {
		td.CmpTrue(t, called, "runTarget was called")
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
	td.CmpError(t, err)
	td.CmpContains(t, err.Error(), "great sadness")

	body, err := os.ReadFile(logfile)
	td.CmpNoError(t, err)
	td.CmpContains(t, string(body), "panic: great sadness")
}
