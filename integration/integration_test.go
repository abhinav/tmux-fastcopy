package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"

	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vito/midterm"
	"go.abhg.dev/io/ioutil"
	"go.uber.org/multierr"
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

type jsonReport struct {
	RegexName    string `json:"regexName"`
	TargetPaneID string `json:"targetPaneID"`
	Text         string `json:"text"`
	CWD          string `json:"cwd"`
}

func jsonReportAction() (exitCode int) {
	txt, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Printf("unable to read stdin: %v", err)
		return 1
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Printf("unable to get cwd: %v", err)
		return 1
	}

	tmuxExe := os.Getenv("TMUX_EXE")
	if len(tmuxExe) == 0 {
		log.Print("TMUX_EXE is unset")
		return 1
	}

	state := jsonReport{
		RegexName:    os.Getenv("FASTCOPY_REGEX_NAME"),
		TargetPaneID: os.Getenv("FASTCOPY_TARGET_PANE_ID"),
		Text:         string(txt),
		CWD:          cwd,
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

//nolint:paralleltest // flaky when parallel
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
		t.Fatalf("could not find EOF\nscreen:\n%s\nraw:\n%s", tmux.Screen(), tmux.Contents())
	}
	time.Sleep(250 * time.Millisecond)

	var matches []matchInfo
	for i := range _wantMatches {
		tmux.Clear()
		fmt.Fprintln(tmux, "clear && cat", testFile)
		require.NoError(t,
			tmux.WaitUntilContains("--EOF--", 3*time.Second),
			"wait for fastcopy window")
		time.Sleep(250 * time.Millisecond)

		base := tmux.Snapshot()
		_, err := tmux.Write([]byte{0x01, 'f'}) // ctrl-a f
		require.NoError(t, err, "send ctrl-a f")

		hints, overlay, err := tmux.WaitUntilHintLabels(base, 3*time.Second)
		require.NoError(t, err, "wait for hint labels")
		t.Logf("got hints %q", hints)
		if !assert.Len(t, hints, len(_wantMatches)) {
			t.Fatalf(
				"expected %d hints\nscreen:\n%s\nraw:\n%s",
				len(_wantMatches),
				overlay,
				tmux.Contents(),
			)
		}

		hint := hints[i]
		t.Logf("selecting %q", hint)
		if shift {
			_, err := io.WriteString(tmux, strings.ToUpper(hint))
			require.NoError(t, err, "select hint")
		} else {
			_, err := io.WriteString(tmux, hint)
			require.NoError(t, err, "select hint")
		}
		time.Sleep(250 * time.Millisecond)

		got, err := tmux.WaitUntilBuffer(5 * time.Second)
		require.NoError(t, err)

		var state jsonReport
		require.NoError(t, json.Unmarshal(got, &state))

		t.Logf("got %+v", state)
		matches = append(matches, matchInfo{
			Regex: state.RegexName,
			Text:  state.Text,
		})
	}

	assert.ElementsMatch(t, _wantMatches, matches)
}

func TestIntegration_ShiftNoop(t *testing.T) {
	t.Parallel()

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
		t.Fatalf("could not find EOF\nscreen:\n%s\nraw:\n%s", tmux.Screen(), tmux.Contents())
	}
	time.Sleep(250 * time.Millisecond)

	tmux.Clear()
	base := tmux.Snapshot()
	_, err := tmux.Write([]byte{0x01, 'f'}) // ctrl-a f
	require.NoError(t, err, "send ctrl-a f")

	hints, overlay, err := tmux.WaitUntilHintLabels(base, 5*time.Second)
	require.NoError(t, err, "wait for hint labels")
	t.Logf("got hints %q", hints)
	require.NotEmpty(t, hints, "expected hints\nscreen:\n%s\nraw:\n%s", overlay, tmux.Contents())

	hint := hints[0]
	t.Logf("selecting %q", hint)
	_, err = io.WriteString(tmux, strings.ToUpper(hint))
	require.NoError(t, err, "select hint")
	time.Sleep(250 * time.Millisecond)

	got, err := tmux.Command("show-buffer").Output()
	if err == nil {
		// The operation may fail because there are no buffers,
		// or it will succeed with an empty buffer.
		assert.Empty(t, string(got), "buffer must be empty")
	}
}

func TestIntegration_ActionEnv(t *testing.T) {
	t.Parallel()

	env := (&fakeEnvConfig{
		Action: "json-report",
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
		Dir:    env.Home,
	}).Build(t)
	time.Sleep(250 * time.Millisecond)
	require.NoError(t, tmux.Command("set-buffer", "").Run(),
		"clear tmux buffer")

	// Create a random directory in the home directory
	// and cd into it.
	cwd := filepath.Join(env.Home, "working-dir")
	require.NoError(t, os.MkdirAll(cwd, 0o755), "create random directory")
	fmt.Fprintln(tmux, "cd", cwd)
	tmux.Clear()

	// Sanity check: make sure we're in the right directory.
	fmt.Fprintln(tmux, "pwd")
	if !assert.NoError(t, tmux.WaitUntilContains(cwd, 5*time.Second)) {
		t.Fatalf("could not find %q\nscreen:\n%s\nraw:\n%s", cwd, tmux.Screen(), tmux.Contents())
	}
	t.Logf("Running fastcopy in %q", cwd)

	// Get the current pane's ID.
	bs, err := tmux.Command("list-panes", "-F", "#{pane_id}").Output()
	require.NoError(t, err)
	targetPaneID := strings.TrimSpace(string(bs))
	require.NotEmpty(t, targetPaneID, "expected pane ID")

	// Clear to ensure the "cat /path/to/whatever" isn't part of the
	// matched text.
	tmux.Clear()
	fmt.Fprintln(tmux, "clear && cat", testFile)
	if !assert.NoError(t, tmux.WaitUntilContains("--EOF--", 5*time.Second)) {
		t.Fatalf("could not find EOF\nscreen:\n%s\nraw:\n%s", tmux.Screen(), tmux.Contents())
	}
	time.Sleep(250 * time.Millisecond)

	tmux.Clear()
	base := tmux.Snapshot()
	_, err = tmux.Write([]byte{0x01, 'f'}) // ctrl-a f
	require.NoError(t, err, "send ctrl-a f")

	hints, overlay, err := tmux.WaitUntilHintLabels(base, 5*time.Second)
	require.NoError(t, err, "wait for hint labels")
	t.Logf("got hints %q", hints)
	require.NotEmpty(t, hints, "expected hints\nscreen:\n%s\nraw:\n%s", overlay, tmux.Contents())

	hint := hints[0]
	t.Logf("selecting %q", hint)

	_, err = io.WriteString(tmux, hint)
	require.NoError(t, err, "select hint")
	time.Sleep(250 * time.Millisecond)

	got, err := tmux.WaitUntilBuffer(5 * time.Second)
	require.NoError(t, err)

	var state jsonReport
	require.NoError(t, json.Unmarshal(got, &state))

	assert.Equal(t, cwd, state.CWD,
		"action directory does not match")
	assert.Equal(t, targetPaneID, state.TargetPaneID,
		"action pane ID does not match")
}

func TestIntegration_MultiSelect(t *testing.T) {
	t.Parallel()

	env := (&fakeEnvConfig{
		Action: "json-report",
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
		t.Fatalf("could not find EOF\nscreen:\n%s\nraw:\n%s", tmux.Screen(), tmux.Contents())
	}
	time.Sleep(250 * time.Millisecond)

	tmux.Clear()
	base := tmux.Snapshot()
	_, err := tmux.Write([]byte{0x01, 'f'}) // ctrl-a f
	require.NoError(t, err, "send ctrl-a f")

	// Enter multi-select mode.
	hints, overlay, err := tmux.WaitUntilHintLabels(base, 5*time.Second)
	require.NoError(t, err, "wait for hint labels")

	_, err = tmux.Write([]byte{0x09})
	require.NoError(t, err, "send tab")

	time.Sleep(200 * time.Millisecond)

	t.Logf("got hints %q", hints)
	if !assert.Len(t, hints, len(_wantMatches)) {
		t.Fatalf(
			"expected %d hints\nscreen:\n%s\nraw:\n%s",
			len(_wantMatches),
			overlay,
			tmux.Contents(),
		)
	}

	// Select all hints.
	for _, hint := range hints {
		_, err := io.WriteString(tmux, hint)
		require.NoError(t, err, "select hint %q", hint)
		time.Sleep(250 * time.Millisecond)
	}

	// Accept output and exit.
	_, err = tmux.Write([]byte{0x0d}) // CR
	require.NoError(t, err, "send enter")

	time.Sleep(250 * time.Millisecond)

	got, err := tmux.WaitUntilBuffer(5 * time.Second)
	require.NoError(t, err)

	var state jsonReport
	require.NoError(t, json.Unmarshal(got, &state))

	t.Logf("got %+v", state)

	// The outputs are space-separated in some order.
	texts := strings.Split(state.Text, " ")
	assert.Len(t, texts, len(_wantMatches), "expected texts to match")

	var wantTexts []string
	for i := range _wantMatches {
		wantTexts = append(wantTexts, _wantMatches[i].Text)
	}

	assert.ElementsMatch(t, wantTexts, texts)
}

func TestIntegration_DestroyUnattached(t *testing.T) {
	t.Parallel()

	env := (&fakeEnvConfig{
		Action:            "json-report",
		DestroyUnattached: true,
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
		Dir:    env.Home,
	}).Build(t)
	time.Sleep(250 * time.Millisecond)
	require.NoError(t, tmux.Command("set-buffer", "").Run(),
		"clear tmux buffer")

	// Clear to ensure the "cat /path/to/whatever" isn't part of the
	// matched text.
	tmux.Clear()
	fmt.Fprintln(tmux, "clear && cat", testFile)
	if !assert.NoError(t, tmux.WaitUntilContains("--EOF--", 5*time.Second)) {
		t.Fatalf("could not find EOF\nscreen:\n%s\nraw:\n%s", tmux.Screen(), tmux.Contents())
	}
	time.Sleep(250 * time.Millisecond)

	tmux.Clear()
	base := tmux.Snapshot()
	_, err := tmux.Write([]byte{0x01, 'f'}) // ctrl-a f
	require.NoError(t, err, "send ctrl-a f")

	hints, overlay, err := tmux.WaitUntilHintLabels(base, 5*time.Second)
	require.NoError(t, err, "wait for hint labels")
	t.Logf("got hints %q", hints)
	require.NotEmpty(t, hints, "expected hints\nscreen:\n%s\nraw:\n%s", overlay, tmux.Contents())

	hint := hints[0]
	t.Logf("selecting %q", hint)

	_, err = io.WriteString(tmux, hint)
	require.NoError(t, err, "select hint")
	time.Sleep(250 * time.Millisecond)

	got, err := tmux.WaitUntilBuffer(5 * time.Second)
	require.NoError(t, err)

	var state jsonReport
	require.NoError(t, json.Unmarshal(got, &state))
	assert.NotEmpty(t, state.Text)
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

	// If true, this will configure tmux to destroy unattached sessions.
	DestroyUnattached bool
}

func (cfg *fakeEnvConfig) Build(t testing.TB) *fakeEnv {
	t.Helper()

	root := mkdirTempGlobal(t, "tmux-fastcopy-integration-")

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
	if cfg.DestroyUnattached {
		cfgLines = append(cfgLines, "set -g destroy-unattached on")
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

	pty *os.File

	mu            sync.RWMutex // guards raw and term
	raw           bytes.Buffer
	rawCheckpoint int
	term          *midterm.Terminal
}

type virtualTmuxConfig struct {
	Tmux          string // path to tmux executable
	Width, Height uint16
	Env           []string
	Dir           string
}

func (cfg *virtualTmuxConfig) Build(t testing.TB) *virtualTmux {
	stderr := ioutil.TestLogWriter(t, "")
	cmd := exec.Command(cfg.Tmux)
	cmd.Env = cfg.Env
	cmd.Stderr = ioutil.TestLogWriter(t, "")
	cmd.Dir = cfg.Dir

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
		term:   midterm.NewTerminal(int(cfg.Height), int(cfg.Width)),
	}

	readerDone := make(chan struct{})
	t.Cleanup(func() {
		assert.NoError(t,
			vt.Command("kill-server").Run(),
			"kill tmux server")
		assert.NoError(t, pty.Close(), "close pty")

		select {
		case <-readerDone:
		case <-time.After(10 * time.Second):
			t.Error("timed out waiting for readLoop to exit")
		}
	})

	go vt.readLoop(pty, readerDone)
	return vt
}

// Command builds an exec.Command for a tmux subcommand.
func (vt *virtualTmux) Command(args ...string) *exec.Cmd {
	cmd := exec.Command(vt.tmux, args...)
	cmd.Env = vt.env
	cmd.Stderr = vt.stderr
	return cmd
}

func (vt *virtualTmux) readLoop(r io.Reader, done chan struct{}) {
	defer close(done)

	bs := make([]byte, 4*1024)
	for {
		n, err := r.Read(bs)
		if err != nil {
			return
		}
		vt.mu.Lock()
		vt.raw.Write(bs[:n])
		_, _ = vt.term.Write(bs[:n])
		vt.mu.Unlock()
	}
}

func (vt *virtualTmux) Write(b []byte) (int, error) {
	return vt.pty.Write(b)
}

func (vt *virtualTmux) Clear() {
	vt.mu.Lock()
	vt.rawCheckpoint = vt.raw.Len()
	vt.mu.Unlock()
}

// Snapshot reports the current decoded terminal contents.
func (vt *virtualTmux) Snapshot() screenSnapshot {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	rows := make([][]rune, len(vt.term.Content))
	for i, row := range vt.term.Content {
		rows[i] = append([]rune(nil), row...)
	}

	return screenSnapshot{rows: rows}
}

// Screen renders the current decoded terminal contents for diagnostics.
func (vt *virtualTmux) Screen() string {
	return vt.Snapshot().String()
}

func (vt *virtualTmux) Contains(s string) bool {
	return vt.Snapshot().Contains(s)
}

func (vt *virtualTmux) Contents() string {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	bs := vt.raw.Bytes()
	return string(bs[vt.rawCheckpoint:])
}

// WaitUntilHintLabels waits for decoded hint labels to appear relative to base.
func (vt *virtualTmux) WaitUntilHintLabels(base screenSnapshot, timeout time.Duration) ([]string, screenSnapshot, error) {
	after := time.After(timeout)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		snapshot := vt.Snapshot()
		labels := snapshot.HintLabels(base)
		if len(labels) > 0 {
			return labels, snapshot, nil
		}

		select {
		case <-after:
			return nil, snapshot, fmt.Errorf("timeout waiting for hint labels")
		case <-ticker.C:
		}
	}
}

// WaitUntilBuffer waits for tmux's current paste buffer to become available.
func (vt *virtualTmux) WaitUntilBuffer(timeout time.Duration) ([]byte, error) {
	after := time.After(timeout)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		bs, err := vt.Command("show-buffer").Output()
		if err == nil {
			return bs, nil
		}

		select {
		case <-after:
			return nil, fmt.Errorf("timeout waiting for tmux buffer")
		case <-ticker.C:
		}
	}
}

func (vt *virtualTmux) WaitUntilContains(str string, timeout time.Duration) error {
	after := time.After(timeout)

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		if vt.Contains(str) {
			return nil
		}

		select {
		case <-after:
			return fmt.Errorf("timeout waiting for %q", str)
		case <-ticker.C:
			// Check again.
		}
	}
}

// screenSnapshot captures the decoded visible contents of the terminal.
type screenSnapshot struct {
	rows [][]rune
}

// String renders the snapshot as newline-separated rows.
func (s screenSnapshot) String() string {
	var out strings.Builder
	for i, row := range s.rows {
		if i > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(string(row))
	}
	return out.String()
}

// Contains reports whether the decoded screen contains the provided text.
func (s screenSnapshot) Contains(str string) bool {
	return strings.Contains(s.String(), str)
}

// HintLabels reports the visible hint labels introduced relative to base.
func (s screenSnapshot) HintLabels(base screenSnapshot) []string {
	var labels []string
	seen := make(map[string]struct{})

	// Compare only the overlapping visible region.
	// The terminal model is fixed-size,
	// but keeping the bounds explicit
	// makes the diff logic safe
	// if either snapshot shape changes in the future.
	height := min(len(base.rows), len(s.rows))

	for y := range height {
		row := s.rows[y]
		baseRow := base.rows[y]

		width := min(len(baseRow), len(row))

		for x := 0; x < width; {
			// A hint label is the new lowercase text
			// overlaid by fastcopy.
			// Skip unchanged cells and non-label cells
			// so we do not accidentally treat pane content
			// as a selectable hint.
			if row[x] == baseRow[x] || !unicode.IsLower(row[x]) {
				x++
				continue
			}

			start := x
			// Consume one contiguous run
			// of overlaid lowercase cells.
			// This preserves the on-screen label order
			// while avoiding any dependency
			// on the original ANSI styling
			// that produced the overlay.
			for x < width && row[x] != baseRow[x] && unicode.IsLower(row[x]) {
				x++
			}

			label := string(row[start:x])
			if len(label) == 0 {
				continue
			}
			if _, ok := seen[label]; ok {
				continue
			}

			// The same label may be drawn more than once
			// when the matched text appears multiple times.
			// Keep only the first visible occurrence
			// so the returned slice matches the order
			// a user would discover on screen.
			labels = append(labels, label)
			seen[label] = struct{}{}
		}
	}

	return labels
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

func copyFile(dst, src string) (err error) {
	i, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer multierr.AppendInvoke(&err, multierr.Close(i))

	o, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("open destination: %w", err)
	}
	defer multierr.AppendInvoke(&err, multierr.Close(o))

	_, err = io.Copy(o, i)
	return err
}

// Creates a temporary directory in /tmp.
// Use this for cases where a long temporary path isn't acceptable.
func mkdirTempGlobal(t testing.TB, pattern string) string {
	t.Helper()

	dir, err := os.MkdirTemp("/tmp", pattern)
	require.NoError(t, err, "create temporary directory")
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(dir), "remove temporary directory")
	})

	dir, err = filepath.EvalSymlinks(dir)
	require.NoError(t, err, "resolve temporary directory")

	return dir
}
