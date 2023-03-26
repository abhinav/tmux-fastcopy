package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/abhinav/tmux-fastcopy/internal/iotest"
	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _behaviors = map[string]func() (exitCode int){
	// Places a JSON report of the regex name and the hint in the buffer.
	"json-report": jsonReportAction,

	// Prints the environment to stderr and exits.
	"unexpected": unexpectedAction,
}

func TestMain(m *testing.M) {
	if b, ok := _behaviors[filepath.Base(os.Args[0])]; ok {
		os.Exit(b())
	}
	os.Exit(m.Run())
}

func behaviorBinary(t testing.TB, name string) string {
	t.Helper()

	_, ok := _behaviors[name]
	require.True(t, ok, "unknown behavior %q", name)

	exe, err := os.Executable()
	require.NoError(t, err, "determine test executable")

	behavior := filepath.Join(t.TempDir(), name)
	require.NoError(t, copyFile(behavior, exe), "copy test executable")
	require.NoError(t, os.Chmod(behavior, 0o755), "mark file executable")
	return behavior
}

func jsonReportAction() (exitCode int) {
	var state struct {
		RegexName string
		Text      string
	}

	txt, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Printf("unable to read stdin: %v", err)
		return 1
	}

	state.RegexName = os.Getenv("FASTCOPY_REGEX_NAME")
	state.Text = string(txt)

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

func unexpectedAction() (exitCode int) {
	log.Print("UNEXPECTED CALL")
	log.Print("  Environment:")
	for _, env := range os.Environ() {
		log.Printf("    %v", env)
	}
	return 1
}

const _giveText = `
IP address: 127.0.0.1
UUID: 95471085-9665-403E-BD95-217C7237F83D
Git SHA: 64b9df0bd2e2709fdd95d4b58ecaf2a1d9e943a7
A line that wraps to the next line: 123456789012345678901234567890123456789012345678901234567890
Phabricator diff: D1234567
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
	{Regex: "phab-diff", Text: "D1234567"},
}

func TestIntegration_SelectMatches(t *testing.T) {
	t.Run("default action", func(t *testing.T) {
		testIntegrationSelectMatches(t, false)
	})

	t.Run("shift action", func(t *testing.T) {
		testIntegrationSelectMatches(t, true)
	})
}

func testIntegrationSelectMatches(t *testing.T, shift bool) {
	var envConfig fakeEnvConfig
	if shift {
		envConfig.Action = "unexpected"
		envConfig.ShiftAction = "json-report"
	} else {
		envConfig.Action = "json-report"
		envConfig.ShiftAction = "unexpected"
	}
	env := envConfig.Build(t)

	testFile := filepath.Join(env.Root, "give.txt")
	require.NoError(t,
		os.WriteFile(testFile, []byte(_giveText), 0o644),
		"write test file")

	tmux := (&virtualTmuxConfig{
		Tmux:   env.Tmux,
		Width:  80,
		Height: 40,
		Env:    env.Environ(),
	}).Build(t)
	time.Sleep(250 * time.Millisecond)
	require.NoError(t, tmux.Command("set-buffer", "").Run(),
		"clear tmux buffer")

	// Clear to ensure the "cat /path/to/whatever" isn't part of the
	// matched text.
	tmux.Clear()
	fmt.Fprintln(tmux, "clear && cat", testFile)
	if !assert.NoError(t, tmux.WaitUntilContains("--EOF--", 5*time.Second)) {
		t.Fatalf("could not find EOF in %q", tmux.Contents())
	}

	var matches []matchInfo
	for i := 0; i < len(_wantMatches); i++ {
		tmux.Clear()
		tmux.Write([]byte{0x01, 'f'}) // ctrl-a f
		time.Sleep(250 * time.Millisecond)
		tmux.WaitUntilContains("--EOF--", 3*time.Second)

		hints := tmux.Hints()
		t.Logf("got hints %q", hints)
		if !assert.Len(t, hints, len(_wantMatches)) {
			t.Fatalf("expected %d hints in %q", len(_wantMatches), tmux.Contents())
		}

		hint := hints[i]
		t.Logf("selecting %q", hint)
		if shift {
			io.WriteString(tmux, strings.ToUpper(hint))
		} else {
			io.WriteString(tmux, hint)
		}
		time.Sleep(250 * time.Millisecond)

		got, err := tmux.Command("show-buffer").Output()
		require.NoError(t, err)

		var state struct {
			RegexName string
			Text      string
		}
		require.NoError(t, json.Unmarshal([]byte(got), &state))

		t.Logf("got %+v", state)
		matches = append(matches, matchInfo{
			Regex: state.RegexName,
			Text:  state.Text,
		})
	}

	assert.ElementsMatch(t, _wantMatches, matches)
}

func TestIntegration_ShiftNoop(t *testing.T) {
	env := (&fakeEnvConfig{
		Action: "unexpected",
	}).Build(t)

	testFile := filepath.Join(env.Root, "give.txt")
	require.NoError(t,
		os.WriteFile(testFile, []byte(_giveText), 0o644),
		"write test file")

	tmux := (&virtualTmuxConfig{
		Tmux:   env.Tmux,
		Width:  80,
		Height: 40,
		Env:    env.Environ(),
	}).Build(t)
	time.Sleep(250 * time.Millisecond)
	require.NoError(t, tmux.Command("set-buffer", "").Run(),
		"clear tmux buffer")

	// Clear to ensure the "cat /path/to/whatever" isn't part of the
	// matched text.
	tmux.Clear()
	fmt.Fprintln(tmux, "clear && cat", testFile)
	if !assert.NoError(t, tmux.WaitUntilContains("--EOF--", 5*time.Second)) {
		t.Fatalf("could not find EOF in %q", tmux.Contents())
	}

	tmux.Clear()
	tmux.Write([]byte{0x01, 'f'}) // ctrl-a f
	require.NoError(t, tmux.WaitUntilContains("--EOF--", 5*time.Second))

	hints := tmux.Hints()
	t.Logf("got hints %q", hints)
	require.NotEmpty(t, hints, "expected hints in %q", tmux.Contents())

	hint := hints[rand.Intn(len(hints))]
	t.Logf("selecting %q", hint)
	io.WriteString(tmux, strings.ToUpper(hint))
	time.Sleep(250 * time.Millisecond)

	got, err := tmux.Command("show-buffer").Output()
	if err == nil {
		// The operation may fail because there are no buffers,
		// or it will succeed with an empty buffer.
		assert.Empty(t, string(got), "buffer must be empty")
	}
}

type fakeEnv struct {
	Root   string
	Home   string
	TmpDir string
	Tmux   string // path to tmux

	coverDir string // $GOCOVERDIR
}

type fakeEnvConfig struct {
	Action      string // name of the behavior
	ShiftAction string // name of the behavior
}

func (cfg *fakeEnvConfig) Build(t testing.TB) *fakeEnv {
	t.Helper()

	root := t.TempDir()

	home := filepath.Join(root, "home")
	require.NoError(t, os.Mkdir(home, 0o1755), "set up home")

	tmpDir := filepath.Join(root, "tmp")
	require.NoError(t, os.Mkdir(tmpDir, 0o1755), "set up tmp")

	binDir := filepath.Join(root, "bin")
	require.NoError(t, os.Mkdir(binDir, 0o1755), "set up bin")

	tmuxFastcopy, err := exec.LookPath("tmux-fastcopy")
	require.NoError(t, err, "find tmux-fastcopy")

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

	cfgLines := []string{
		"set -g prefix C-a",
		"set -g status off",
		fmt.Sprintf("set -g default-shell %v", bash),
		fmt.Sprintf("bind-key f run-shell -b %q", fastcopyCmd),
		`set -g @fastcopy-regex-phab-diff '\bD\d{3,}\b'`, // custom regex
	}
	if len(cfg.Action) > 0 {
		exe := behaviorBinary(t, cfg.Action)
		cfgLines = append(cfgLines, fmt.Sprintf("set -g @fastcopy-action %q", exe))
	}
	if len(cfg.ShiftAction) > 0 {
		exe := behaviorBinary(t, cfg.ShiftAction)
		cfgLines = append(cfgLines, fmt.Sprintf("set -g @fastcopy-shift-action %q", exe))
	}

	writeLines(t, filepath.Join(home, ".tmux.conf"), cfgLines...)
	t.Logf("Using tmux config:\n%s", strings.Join(cfgLines, "\n"))

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
		coverDir: os.Getenv("GOCOVERDIR"),
	}
}

func (e *fakeEnv) Environ() []string {
	return []string{
		"HOME=" + e.Home,
		"TERM=screen",
		"TMUX_TMPDIR=" + e.TmpDir,
		"TMUX_EXE=" + e.Tmux,
		"GOCOVERDIR=" + e.coverDir,
	}
}

// virtualTmux is a tmux session that can be used to test tmux-fastcopy.
type virtualTmux struct {
	tmux   string
	env    []string
	w, h   int
	stderr io.Writer

	pty  *os.File
	mu   sync.RWMutex // guards buff
	buff bytes.Buffer // output
}

type virtualTmuxConfig struct {
	Tmux          string // path to tmux executable
	Width, Height uint16
	Env           []string
}

func (cfg *virtualTmuxConfig) Build(t testing.TB) *virtualTmux {
	stderr := iotest.Writer(t)
	cmd := exec.Command(cfg.Tmux)
	cmd.Env = cfg.Env
	cmd.Stderr = iotest.Writer(t)

	t.Logf("Starting tmux with size %dx%d", cfg.Width, cfg.Height)
	pty, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: cfg.Height,
		Cols: cfg.Width,
	})
	require.NoError(t, err, "start tmux")

	vt := &virtualTmux{
		w:      int(cfg.Width),
		h:      int(cfg.Height),
		tmux:   cfg.Tmux,
		env:    cfg.Env,
		pty:    pty,
		stderr: stderr,
	}

	readerDone := make(chan struct{})
	t.Cleanup(func() {
		vt.Command("kill-server").Run()
		pty.Close()

		select {
		case <-readerDone:
		case <-time.After(10 * time.Second):
			t.Error("timed out waiting for readLoop to exit")
		}
	})

	go vt.readLoop(t, pty, readerDone)
	return vt
}

// Command builds an exec.Command for a tmux subcommand.
func (vt *virtualTmux) Command(args ...string) *exec.Cmd {
	cmd := exec.Command(vt.tmux, args...)
	cmd.Env = vt.env
	cmd.Stderr = vt.stderr
	return cmd
}

func (vt *virtualTmux) readLoop(t testing.TB, r io.Reader, done chan struct{}) {
	defer close(done)

	bs := make([]byte, 4*1024)
	for {
		n, err := r.Read(bs)
		if err != nil {
			return
		}
		vt.mu.Lock()
		vt.buff.Write(bs[:n])
		vt.mu.Unlock()
	}
}

func (vt *virtualTmux) Write(b []byte) (int, error) {
	return vt.pty.Write(b)
}

func (vt *virtualTmux) Clear() {
	// TODO: Auto clear on escape sequence
	vt.mu.Lock()
	vt.buff.Reset()
	vt.mu.Unlock()
}

func (vt *virtualTmux) Contains(s string) bool {
	bs := []byte(s)

	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return bytes.Contains(vt.buff.Bytes(), bs)
}

func (vt *virtualTmux) Contents() string {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.buff.String()
}

var _redText = [][]byte{
	[]byte("\x1b[91m"),
	[]byte("\x1b[31m"),
}

func (vt *virtualTmux) Hints() []string {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	bs := vt.buff.Bytes()
	var hints []string
	seen := make(map[string]struct{})
	for {
		idx := bytes.IndexByte(bs, 0x1b)
		if idx == -1 {
			break // no more hints
		}
		bs = bs[idx:]

		found := false
		for _, red := range _redText {
			if bytes.HasPrefix(bs, red) {
				bs = bs[len(red):]
				found = true
				break
			}
		}
		if !found {
			bs = bs[1:]
			continue // not red text; skip over the escape
		}

		end := bytes.IndexByte(bs, 0x1b)
		if end == -1 {
			break
		}
		hint := string(bs[:end])
		if _, ok := seen[hint]; !ok {
			hints = append(hints, hint)
			seen[hint] = struct{}{}
		}
		bs = bs[end+1:]
	}
	return hints
}

func (vt *virtualTmux) WaitUntilContains(str string, timeout time.Duration) error {
	bs := []byte(str)

	after := time.After(timeout)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		vt.mu.RLock()
		if bytes.Contains(vt.buff.Bytes(), bs) {
			vt.mu.RUnlock()
			return nil
		}
		vt.mu.RUnlock()

		select {
		case <-after:
			return fmt.Errorf("timeout waiting for %q", str)
		case <-ticker.C:
			// Check again.
		}
	}
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
