package tmux

import (
	"crypto/rand"
	"os/exec"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestShowOptions_Args(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give ShowOptionsRequest
		want []string
	}{
		{
			desc: "empty",
			want: []string{"show-options"},
		},
		{
			desc: "global",
			give: ShowOptionsRequest{Global: true},
			want: []string{"show-options", "-g"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			blob := make([]byte, 10)
			rand.Read(blob)

			called := false
			defer func() {
				td.CmpTrue(t, called, "runner.Output not invoked")
			}()
			run := runner{
				Output: func(cmd *exec.Cmd) ([]byte, error) {
					called = true
					td.Cmp(t, cmd.Args[1:], tt.want)
					return blob, nil
				},
			}

			driver := ShellDriver{run: &run}
			got, err := driver.ShowOptions(tt.give)
			td.CmpNoError(t, err)
			td.Cmp(t, got, blob)
		})
	}
}
