package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/fastcopy"
	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommandAction(t *testing.T) {
	t.Parallel()

	cwd := "/foo/bar"

	tests := []struct {
		desc string
		give newActionRequest

		wantArg   *argAction
		wantStdin *stdinAction
		wantErr   string
	}{
		{
			desc:    "empty",
			give:    newActionRequest{Action: ""},
			wantErr: `empty action`,
		},
		{
			desc:    "parse error",
			give:    newActionRequest{Action: `foo "`},
			wantErr: `invalid command line string`,
		},
		{
			desc: "stdin",
			give: newActionRequest{Action: "pbcopy"},
			wantStdin: &stdinAction{
				Cmd:  "pbcopy",
				Args: []string{},
				Dir:  cwd,
			},
		},
		{
			desc: "argument",
			give: newActionRequest{Action: "tmux set-buffer -- {}"},
			wantArg: &argAction{
				Cmd:        "tmux",
				BeforeArgs: []string{"set-buffer", "--"},
				AfterArgs:  []string{},
				Dir:        cwd,
			},
		},
		{
			desc: "stdin with dir",
			give: newActionRequest{
				Action: "pbcopy",
				Dir:    "/tmp",
			},
			wantStdin: &stdinAction{
				Cmd:  "pbcopy",
				Args: []string{},
				Dir:  "/tmp",
			},
		},
		{
			desc: "argument with dir",
			give: newActionRequest{
				Action: "tmux set-buffer -- {}",
				Dir:    "/tmp",
			},
			wantArg: &argAction{
				Cmd:        "tmux",
				BeforeArgs: []string{"set-buffer", "--"},
				AfterArgs:  []string{},
				Dir:        "/tmp",
			},
		},
		{
			desc: "stdin with pane ID",
			give: newActionRequest{
				Action:       "pbcopy",
				TargetPaneID: "123",
			},
			wantStdin: &stdinAction{
				Cmd:    "pbcopy",
				Args:   []string{},
				PaneID: "123",
				Dir:    cwd,
			},
		},
		{
			desc: "argument with pane ID",
			give: newActionRequest{
				Action:       "tmux set-buffer -- {}",
				TargetPaneID: "123",
			},
			wantArg: &argAction{
				Cmd:        "tmux",
				BeforeArgs: []string{"set-buffer", "--"},
				AfterArgs:  []string{},
				PaneID:     "123",
				Dir:        cwd,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			got, err := (&actionFactory{
				Getwd: func() (string, error) {
					return cwd, nil
				},
			}).New(tt.give)

			switch {
			case len(tt.wantErr) > 0:
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)

			case tt.wantArg != nil:
				require.NoError(t, err)
				assert.Equal(t, tt.wantArg, got)

			case tt.wantStdin != nil:
				require.NoError(t, err)
				assert.Equal(t, tt.wantStdin, got)

			default:
				assert.FailNow(t, "invalid test case")
			}
		})
	}
}

func TestNewCommandAction_noCWD(t *testing.T) {
	t.Parallel()

	got, err := (&actionFactory{
		Getwd: func() (string, error) {
			return "", errors.New("great sadness")
		},
	}).New(newActionRequest{Action: "pbcopy"})
	require.NoError(t, err)

	assert.Empty(t, got.(*stdinAction).Dir)
}

func TestStdinAction(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer

	action := stdinAction{
		Cmd:     "cat",
		Log:     log.New(&buff),
		Environ: func() []string { return nil },
	}
	require.NoError(t, action.Run(fastcopy.Selection{
		Text:     "foo",
		Matchers: []string{"x"},
	}))
	assert.Equal(t, "[cat] foo\n", buff.String())
}

func TestStdinAction_RegexesEnv(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer

	action := stdinAction{
		Cmd: "env",
		Log: log.New(&buff),
		Environ: func() []string {
			return []string{"FOO=bar"}
		},
	}
	require.NoError(t, action.Run(fastcopy.Selection{
		Text:     "foo",
		Matchers: []string{"x", "y"},
	}))
	assert.Contains(t, buff.String(), "[env] FASTCOPY_REGEX_NAME=x y\n")
	assert.Contains(t, buff.String(), "[env] FOO=bar\n")
}

func TestArgAction(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	action := argAction{
		Cmd:        "echo",
		BeforeArgs: []string{"1", "2"},
		AfterArgs:  []string{"3", "4"},
		Log:        log.New(&buff),
		Environ:    func() []string { return nil },
	}
	require.NoError(t, action.Run(fastcopy.Selection{
		Text:     "foo",
		Matchers: []string{"x"},
	}))
	assert.Equal(t, "[echo] 1 2 foo 3 4\n", buff.String())
}

func TestArgAction_RegexesEnv(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	action := argAction{
		Cmd:        "bash",
		BeforeArgs: []string{"-c", "env"},
		Log:        log.New(&buff),
		Environ: func() []string {
			return []string{"FOO=bar"}
		},
	}
	require.NoError(t, action.Run(fastcopy.Selection{
		Text:     "foo",
		Matchers: []string{"x", "y"},
	}))
	assert.Contains(t, buff.String(), "[bash] FASTCOPY_REGEX_NAME=x y\n")
	assert.Contains(t, buff.String(), "[bash] FOO=bar\n")
}
