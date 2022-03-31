package main

import (
	"bytes"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/fastcopy"
	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommandAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give string

		wantArg   *argAction
		wantStdin *stdinAction
		wantErr   string
	}{
		{
			desc:    "empty",
			give:    "",
			wantErr: `empty action`,
		},
		{
			desc:    "parse error",
			give:    `foo "`,
			wantErr: `invalid command line string`,
		},
		{
			desc: "stdin",
			give: "pbcopy",
			wantStdin: &stdinAction{
				Cmd:  "pbcopy",
				Args: []string{},
			},
		},
		{
			desc: "argument",
			give: "tmux set-buffer -- {}",
			wantArg: &argAction{
				Cmd:        "tmux",
				BeforeArgs: []string{"set-buffer", "--"},
				AfterArgs:  []string{},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			got, err := new(actionFactory).New(tt.give)
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

func TestStdinAction(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer

	action := stdinAction{
		Cmd: "cat",
		Log: log.New(&buff),
	}
	require.NoError(t, action.Run(fastcopy.Selection{
		Text:     "foo",
		Matchers: []string{"x"},
	}))
	assert.Equal(t, "[cat] foo\n", buff.String())
}

func TestArgAction(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	action := argAction{
		Cmd:        "echo",
		BeforeArgs: []string{"1", "2"},
		AfterArgs:  []string{"3", "4"},
		Log:        log.New(&buff),
	}
	require.NoError(t, action.Run(fastcopy.Selection{
		Text:     "foo",
		Matchers: []string{"x"},
	}))
	assert.Equal(t, "[echo] 1 2 foo 3 4\n", buff.String())
}
