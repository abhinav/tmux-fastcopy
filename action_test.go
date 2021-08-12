package main

import (
	"bytes"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/log"
	"github.com/maxatome/go-testdeep/td"
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
				td.CmpError(t, err)
				td.CmpContains(t, err.Error(), tt.wantErr)

			case tt.wantArg != nil:
				td.CmpNoError(t, err)
				td.Cmp(t, got, tt.wantArg)

			case tt.wantStdin != nil:
				td.CmpNoError(t, err)
				td.Cmp(t, got, tt.wantStdin)

			default:
				t.Fatal("invalid test case")
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
	td.CmpNoError(t, action.Run("foo"))
	td.Cmp(t, buff.String(), "[cat] foo\n")
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
	td.CmpNoError(t, action.Run("foo"))
	td.Cmp(t, buff.String(), "[echo] 1 2 foo 3 4\n")
}
