package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/abhinav/tmux-fastcopy/internal/iotest"
	"github.com/creack/pty"
	"github.com/jaguilar/vt100"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const _giveText = `
IP address: 127.0.0.1
UUID: 95471085-9665-403E-BD95-217C7237F83D
Git SHA: 64b9df0bd2e2709fdd95d4b58ecaf2a1d9e943a7
A line that wraps to the next line: 123456789012345678901234567890123456789012345678901234567890
--EOF--
`

var _wantMatches = []string{
	"127.0.0.1",
	"95471085-9665-403E-BD95-217C7237F83D",
	"64b9df0bd2e2709fdd95d4b58ecaf2a1d9e943a7",
	"123456789012345678901234567890123456789012345678901234567890",
}

func TestIntegration(t *testing.T) {
	env := newFakeEnv(t)

	testFile := filepath.Join(env.Root, "give.txt")
	require.NoError(t,
		os.WriteFile(testFile, []byte(_giveText), 0644),
		"write test file")

	tmux := (&vTmuxConfig{
		Tmux:   env.Tmux,
		Width:  80,
		Height: 40,
		Env:    env.Environ(),
	}).Build(t)

	time.Sleep(time.Second)

	// Clear to ensure the "cat /path/to/whatever" isn't part of the
	// matched text.
	fmt.Fprintln(tmux, "clear && cat", testFile)
	if !assert.NoError(t, tmux.WaitUntilContains("--EOF--", 5*time.Second)) {
		t.Fatalf("could not find EOF in %q", tmux.Lines())
	}

	var matches []string
	for i := 0; i < len(_wantMatches); i++ {
		tmux.Write([]byte{0x01, 'f'}) // ctrl-a f
		time.Sleep(500 * time.Millisecond)
		tmux.WaitUntilContains("--EOF--", 3*time.Second)

		hints := tmux.Matches(func(_ rune, f vt100.Format) bool {
			return f.Fg == vt100.Red
		})
		t.Logf("got hints %q", hints)
		if !assert.Len(t, hints, len(_wantMatches)) {
			t.Fatalf("expected %d hints in %q", len(_wantMatches), tmux.Lines())
		}

		hint := hints[i]
		t.Logf("selecting %q", hint)
		io.WriteString(tmux, hint)
		time.Sleep(100 * time.Millisecond)

		got, err := tmux.Command("show-buffer").Output()
		require.NoError(t, err)

		t.Logf("got %q", got)
		matches = append(matches, string(got))
	}

	assert.ElementsMatch(t, _wantMatches, matches)
}

type fakeEnv struct {
	Root   string
	Home   string
	TmpDir string
	Tmux   string // path to tmux
}

func newFakeEnv(t testing.TB) *fakeEnv {
	t.Helper()

	root := t.TempDir()

	home := filepath.Join(root, "home")
	require.NoError(t, os.Mkdir(home, 01755), "set up home")

	tmpDir := filepath.Join(root, "tmp")
	require.NoError(t, os.Mkdir(tmpDir, 01755), "set up tmpDir")

	logFile := filepath.Join(root, "log.txt")
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}

		got, err := os.ReadFile(logFile)
		if err != nil {
			t.Logf("unable to read log %q: %v", logFile, err)
		} else {
			t.Logf("tmux-fastcopy log:\n%s", got)
		}
	})

	tmux, err := exec.LookPath("tmux")
	require.NoError(t, err, "find tmux")

	tmuxFastcopy, err := exec.LookPath("tmux-fastcopy")
	require.NoError(t, err, "find tmux-fastcopy")

	bash, err := exec.LookPath("bash")
	require.NoError(t, err, "find bash")

	fastcopyCmd := fmt.Sprintf("%v --verbose --log %v --tmux %v",
		tmuxFastcopy, logFile, tmux)

	writeLines(t, filepath.Join(home, ".tmux.conf"),
		"set -g prefix C-a",
		"set -g status off",
		fmt.Sprintf("set -g default-shell %v", bash),
		fmt.Sprintf("bind-key f run-shell -b %q", fastcopyCmd),
	)

	writeLines(t, filepath.Join(home, ".bash_profile"),
		`export PS1="$ "`, // minimal prompt
	)

	out, err := exec.Command(tmux, "-V").Output()
	require.NoError(t, err)
	t.Logf("using %s", out)

	return &fakeEnv{
		Root:   root,
		Home:   home,
		TmpDir: tmpDir,
		Tmux:   tmux,
	}
}

func (e *fakeEnv) Environ() []string {
	return []string{
		"HOME=" + e.Home,
		"TERM=xterm-256color",
		"TMUX_TMPDIR=" + e.TmpDir,
	}
}

// Virtual controllable tmux.
type vTmux struct {
	tmux string
	env  []string
	w, h int
	pty  *os.File
	mu   sync.RWMutex // guards vt
	vt   *vt100.VT100
}

type vTmuxConfig struct {
	Tmux          string // path to tmux executable
	Width, Height uint16
	Env           []string
}

func (cfg *vTmuxConfig) Build(t testing.TB) *vTmux {
	cmd := exec.Command(cfg.Tmux)
	cmd.Env = cfg.Env
	cmd.Stderr = iotest.Writer(t)

	pty, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: cfg.Height,
		Cols: cfg.Width,
	})
	require.NoError(t, err, "start tmux")

	vt := &vTmux{
		w:    int(cfg.Width),
		h:    int(cfg.Height),
		tmux: cfg.Tmux,
		env:  cfg.Env,
		vt:   vt100.NewVT100(int(cfg.Height), int(cfg.Width)),
		pty:  pty,
	}
	t.Cleanup(func() {
		vt.Command("kill-server").Run()
	})

	go vt.start(t, bufio.NewReader(pty))
	return vt
}

func (vt *vTmux) Command(args ...string) *exec.Cmd {
	cmd := exec.Command(vt.tmux, args...)
	cmd.Env = vt.env
	return cmd
}

func (vt *vTmux) start(t testing.TB, vr *bufio.Reader) {
	for {
		cmd, err := vt100.Decode(vr)
		if err == nil {
			vt.mu.Lock()
			err = vt.vt.Process(cmd)
			vt.mu.Unlock()
		}

		switch {
		case err == nil, errors.As(err, new(vt100.UnsupportedError)):
			// ignore unsupported operations
			continue

		case errors.Is(err, io.EOF), errors.Is(err, fs.ErrClosed):
			return

		default:
			// Not an error because this could be an error from
			// attempting to read from a file after it's been
			// cleaned up. Matching on EOF/ErrClosed isn't enough,
			// apparently.
			t.Logf("error decoding vt100 command: %v", err)
			return
		}
	}
}

func (vt *vTmux) Write(b []byte) (int, error) {
	return vt.pty.Write(b)
}

func (vt *vTmux) Contains(s string) bool {
	rs := []rune(s)

	vt.mu.RLock()
	defer vt.mu.RUnlock()

	for row := 0; row < vt.h; row++ {
		i := 0
		for col := 0; col < vt.w; col++ {
			if vt.vt.Content[row][col] == rs[i] {
				i++
			} else {
				i = 0
			}
			if i == len(rs) {
				return true
			}
		}
	}
	return false
}

func (vt *vTmux) Lines() []string {
	lines := make([]string, 0, vt.h)
	line := make([]rune, 0, vt.w)

	vt.mu.RLock()
	defer vt.mu.RUnlock()
	for row := 0; row < vt.h; row++ {
		line = append(line[:0], vt.vt.Content[row]...)
		if l := strings.TrimSpace(string(line)); len(l) > 0 {
			lines = append(lines, l)
		}
	}

	return lines
}

// Find contiguous strings that match this predicate.
func (vt *vTmux) Matches(want func(rune, vt100.Format) bool) []string {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	var matches []string

	for row := 0; row < vt.h; row++ {
		var match []rune
		for col := 0; col < vt.h; col++ {
			r := vt.vt.Content[row][col]
			f := vt.vt.Format[row][col]

			if want(r, f) {
				match = append(match, r)
			} else if len(match) > 0 {
				matches = append(matches, string(match))
				match = match[:0]
			}
		}
		if len(match) > 0 {
			matches = append(matches, string(match))
		}
	}

	return matches
}

func (vt *vTmux) WaitUntilContains(str string, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		if vt.Contains(str) {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("could not find %q in %v", str, timeout)
}

func writeLines(t testing.TB, path string, lines ...string) {
	t.Helper()

	f, err := os.Create(path)
	require.NoError(t, err, "create %q", path)
	defer func() {
		assert.NoError(t, f.Close(), "close %q", path)
	}()

	for _, line := range lines {
		fmt.Fprintln(f, line)
	}
}
