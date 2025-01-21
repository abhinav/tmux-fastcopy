package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxopt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.abhg.dev/io/ioutil"
)

func TestUnquoteTmuxOptions(t *testing.T) {
	t.Parallel()

	tmuxExe, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux not found in PATH")
	}

	root := mkdirTempGlobal(t, "tmux-fastcopy-unquote-test")

	home := filepath.Join(root, "home")
	require.NoError(t, os.Mkdir(home, 0o755))

	tmpDir := filepath.Join(root, "tmp")
	require.NoError(t, os.Mkdir(tmpDir, 0o755))

	cfgFile := filepath.Join(home, ".tmux.conf")
	require.NoError(t, os.WriteFile(cfgFile, []byte(`set -g exit-empty off`), 0o644))

	env := []string{
		"HOME=" + home,
		"TERM=screen",
		"SHELL=/bin/sh",
		"TMUX_TMPDIR=" + tmpDir,
	}
	cmdout := ioutil.TestLogWriter(t, "")

	cmd := exec.Command(tmuxExe, "start-server", ";", "new-session", "-d")
	cmd.Dir = root
	cmd.Env = env
	cmd.Stdout = cmdout
	cmd.Stderr = cmdout
	require.NoError(t, cmd.Run())
	t.Cleanup(func() {
		cmd := exec.Command(tmuxExe, "kill-server")
		cmd.Dir = root
		cmd.Env = env
		cmd.Stdout = cmdout
		cmd.Stderr = cmdout
		assert.NoError(t, cmd.Run())
	})

	tests := []struct {
		name string
		give string // string to set and expect back
	}{
		{"empty", ""},
		{"simple", "foo bar"},
		{"single quote", "foo 'bar'"},
		{"double quote", `foo "bar"`},
		{"escape", `foo "bar\"baz"`},
		{"escape/single quote", `foo 'bar\"baz'`},
		{"escape/double quote", `foo "bar\"baz"`},
		{"escape/escape", `foo "bar\\baz"`},
		{"regex", `(\b([\w.-]+|~)?(/[\w.-]+)+\b)`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			optName := "@" + strings.Map(func(r rune) rune {
				switch r {
				case ' ', '/':
					return '-'
				}
				return r
			}, tt.name)

			cmd := exec.Command(tmuxExe, "set-option", optName, tt.give)
			cmd.Dir = root
			cmd.Env = env
			cmd.Stdout = cmdout
			cmd.Stderr = cmdout
			require.NoError(t, cmd.Run())

			cmd = exec.Command(tmuxExe, "show-option", optName)
			cmd.Dir = root
			cmd.Env = env
			cmd.Stderr = cmdout
			bs, err := cmd.Output()
			require.NoError(t, err)
			bs = bytes.TrimSuffix(bs, []byte{'\n'}) // tmux appends a newline

			name, value, ok := bytes.Cut(bs, []byte{' '})
			require.True(t, ok, "invalid output from tmux show-options: %q", bs)

			assert.Equal(t, optName, string(name), "got back wrong optoin")

			got := tmuxopt.Unquote(value)
			assert.Equal(t, tt.give, string(got), "parsed wrong value")
		})
	}
}
