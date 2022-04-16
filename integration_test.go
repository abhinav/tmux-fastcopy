package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/abhinav/tmux-fastcopy/internal/coverage"
	"github.com/abhinav/tmux-fastcopy/internal/iotest"
	"github.com/creack/pty"
	"github.com/jaguilar/vt100"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	switch filepath.Base(os.Args[0]) {
	case "tmux-fastcopy":
		os.Exit(fakeMain())
	case "test-action":
		os.Exit(fakeAction())
	default:
		os.Exit(m.Run())
	}
}

const _integrationTestCoverDirKey = "TMUX_FASTCOPY_INTEGRATION_TEST_COVER_DIR"

func fakeMain() (exitCode int) {
	if coverDir := os.Getenv(_integrationTestCoverDirKey); len(coverDir) > 0 {
		f, err := os.CreateTemp(coverDir, "tmux-fastcopy-cover")
		if err != nil {
			log.Printf("cannot create coverage file: %v", err)
			return 1
		}
		if err := f.Close(); err != nil {
			log.Printf("cannot close file: %v", err)
			return 1
		}

		defer func() {
			if err := coverage.Report(f.Name()); err != nil {
				log.Printf("cannot report coverage: %v", err)
				exitCode = 1
			}
		}()
	}

	err := run(&_main, os.Args[1:])
	if err != nil && err != flag.ErrHelp {
		fmt.Fprintln(_main.Stderr, err)
		return 1
	}
	return 0
}

func fakeAction() (exitCode int) {
	var state struct {
		RegexName string
		Shift     bool
		Text      string
	}

	flag.BoolVar(&state.Shift, "shift", false, "whether shift is pressed")
	flag.Parse()

	if flag.NArg() == 0 {
		log.Print("expected match as an argument")
		return 1
	}

	state.RegexName = os.Getenv("FASTCOPY_REGEX_NAME")
	state.Text = flag.Arg(0)

	tmuxExe := os.Getenv("TMUX_EXE")
	if len(tmuxExe) == 0 {
		log.Print("TMUX_EXE is unset")
		return 1
	}

	bs, err := json.Marshal(state)
	if err != nil {
		log.Printf("cannot marshal %v: %v", state, err)
		return 1
	}

	cmd := exec.Command(tmuxExe, "load-buffer", "-")
	cmd.Stdin = bytes.NewReader(bs)
	if err := cmd.Run(); err != nil {
		log.Printf("failed to write to tmux buffer: %v", err)
		return 1
	}

	return 0
}

const _giveText = `
IP address: 127.0.0.1
UUID: 95471085-9665-403E-BD95-217C7237F83D
Git SHA: 64b9df0bd2e2709fdd95d4b58ecaf2a1d9e943a7
A line that wraps to the next line: 123456789012345678901234567890123456789012345678901234567890
--EOF--
`

type matchInfo struct {
	Regex string
	Text  string
}

var _wantMatches = []matchInfo{
	{Regex: "ipv4", Text: "127.0.0.1"},
	{Regex: "uuid", Text: "95471085-9665-403E-BD95-217C7237F83D"},
	{Regex: "gitsha", Text: "64b9df0bd2e2709fdd95d4b58ecaf2a1d9e943a7"},
	{Regex: "int", Text: "123456789012345678901234567890123456789012345678901234567890"},
}

func TestIntegration(t *testing.T) {
	env := newFakeEnv(t)

	testFile := filepath.Join(env.Root, "give.txt")
	require.NoError(t,
		os.WriteFile(testFile, []byte(_giveText), 0o644),
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

	var matches []matchInfo
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

		var state struct {
			RegexName string
			Shift     bool
			Text      string
		}
		require.NoError(t, json.Unmarshal([]byte(got), &state))

		t.Logf("got %+v", state)
		assert.False(t, state.Shift, "shift must not be pressed for %v", state)
		matches = append(matches, matchInfo{
			Regex: state.RegexName,
			Text:  state.Text,
		})
	}

	assert.ElementsMatch(t, _wantMatches, matches)

	t.Run("shift", func(t *testing.T) {
		tmux.Write([]byte{0x01, 'f'}) // ctrl-a f
		time.Sleep(500 * time.Millisecond)
		tmux.WaitUntilContains("--EOF--", 3*time.Second)

		hints := tmux.Matches(func(_ rune, f vt100.Format) bool {
			return f.Fg == vt100.Red
		})
		t.Logf("got hints %q", hints)

		hint := hints[rand.Intn(len(hints))]
		t.Logf("selecting %q", hint)
		io.WriteString(tmux, strings.ToUpper(hint))
		time.Sleep(100 * time.Millisecond)

		got, err := tmux.Command("show-buffer").Output()
		require.NoError(t, err)

		var state struct {
			RegexName string
			Shift     bool
			Text      string
		}
		require.NoError(t, json.Unmarshal([]byte(got), &state))
		t.Logf("got %+v", state)
		assert.True(t, state.Shift, "shift must be pressed for %v", state)
	})
}

type fakeEnv struct {
	Root   string
	Home   string
	TmpDir string
	Tmux   string // path to tmux

	coverDir string
}

func newFakeEnv(t testing.TB) *fakeEnv {
	t.Helper()

	root := t.TempDir()

	home := filepath.Join(root, "home")
	require.NoError(t, os.Mkdir(home, 0o1755), "set up home")

	tmpDir := filepath.Join(root, "tmp")
	require.NoError(t, os.Mkdir(tmpDir, 0o1755), "set up tmp")

	binDir := filepath.Join(root, "bin")
	require.NoError(t, os.Mkdir(binDir, 0o1755), "set up bin")

	testExe, err := os.Executable()
	require.NoError(t, err, "determine test executable")

	tmuxFastcopy := filepath.Join(binDir, "tmux-fastcopy")
	require.NoError(t, copyFile(tmuxFastcopy, testExe), "copy test executable")
	require.NoError(t, os.Chmod(tmuxFastcopy, 0o755), "mark tmux-fastcopy as executable")

	fakeAction := filepath.Join(binDir, "test-action")
	require.NoError(t, copyFile(fakeAction, testExe), "copy action executable")
	require.NoError(t, os.Chmod(fakeAction, 0o755), "mark action as executable")

	coverBucket, err := coverage.NewBucket(testing.CoverMode())
	require.NoError(t, err, "failed to set up coverage bucket")
	t.Cleanup(func() {
		assert.NoError(t, coverBucket.Finalize(), "could not finalize coverage")
	})

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

	bash, err := exec.LookPath("bash")
	require.NoError(t, err, "find bash")

	fastcopyCmd := fmt.Sprintf("%v --verbose --log %v --tmux %v",
		tmuxFastcopy, logFile, tmux)

	writeLines(t, filepath.Join(home, ".tmux.conf"),
		"set -g prefix C-a",
		"set -g status off",
		fmt.Sprintf("set -g default-shell %v", bash),
		fmt.Sprintf("bind-key f run-shell -b %q", fastcopyCmd),
		fmt.Sprintf("set -g @fastcopy-action '%v {}'", fakeAction),
		fmt.Sprintf("set -g @fastcopy-shift-action '%v -shift {}'", fakeAction),
	)

	writeLines(t, filepath.Join(home, ".bash_profile"),
		`export PS1="$ "`, // minimal prompt
	)

	out, err := exec.Command(tmux, "-V").Output()
	require.NoError(t, err)
	t.Logf("using %s", out)

	return &fakeEnv{
		Root:     root,
		Home:     home,
		TmpDir:   tmpDir,
		Tmux:     tmux,
		coverDir: coverBucket.Dir(),
	}
}

func (e *fakeEnv) Environ() []string {
	return []string{
		"HOME=" + e.Home,
		"TERM=xterm-256color",
		"TMUX_TMPDIR=" + e.TmpDir,
		_integrationTestCoverDirKey + "=" + e.coverDir,
		"TMUX_EXE=" + e.Tmux,
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

		case errors.Is(err, io.EOF),
			errors.Is(err, fs.ErrClosed),
			errors.Is(err, syscall.EIO):
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

func copyFile(dst, src string) error {
	i, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer i.Close()

	o, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("open destination: %w", err)
	}
	defer o.Close()

	_, err = io.Copy(o, i)
	return err
}
