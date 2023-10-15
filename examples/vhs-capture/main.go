// vhs-capture runs the vhs tool to record one or more gifs from .tape files.
//
// It runs each tape in an isolated environment with a bare tmux configuration
// and a .tmux.conf configured to invoke tmux-fastcopy.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

var _concurrency = flag.Int("c", runtime.GOMAXPROCS(0), "number of tapes to capture concurrently")

func main() {
	log.SetFlags(0)
	flag.Parse()

	if flag.NArg() == 0 {
		log.Fatal("vhs-capture: no tapes specified")
	}

	if err := run(flag.Args()); err != nil {
		log.Fatal("vhs-capture:", err)
	}
}

func run(tapes []string) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}

	tmuxFastcopy, err := exec.LookPath("tmux-fastcopy")
	if err != nil {
		return fmt.Errorf("tmux-fastcopy not found in PATH: %w", err)
	}

	bash, err := exec.LookPath("bash")
	if err != nil {
		return fmt.Errorf("bash not found in PATH: %w", err)
	}

	var (
		wg      sync.WaitGroup
		workers []*worker
	)
	wg.Add(*_concurrency)
	tapec := make(chan string)
	for i := 0; i < *_concurrency; i++ {
		w := &worker{
			Done:         wg.Done,
			TapeC:        tapec,
			Bash:         bash,
			TmuxFastcopy: tmuxFastcopy,
			WorkDir:      workDir,
		}
		go w.Run()
		workers = append(workers, w)
	}

	for _, tape := range tapes {
		tapec <- tape
	}
	close(tapec)
	wg.Wait()

	var errs []error
	for _, w := range workers {
		errs = append(errs, w.Errors...)
	}
	return errors.Join(errs...)
}

type worker struct {
	Done  func()
	TapeC <-chan string

	Bash         string // path to bash
	TmuxFastcopy string // path to tmux-fastcopy
	WorkDir      string // path to work directory

	Errors []error
}

func (w *worker) Run() {
	defer w.Done()

	for tape := range w.TapeC {
		if err := w.run(tape); err != nil {
			w.Errors = append(w.Errors, fmt.Errorf("%s: %v", tape, err))
		}
	}
}

const _bashProfile = `
export PS1="$ "
`

func (w *worker) run(tapeCfg string) error {
	vhsTape, err := loadVHSTape(tapeCfg)
	if err != nil {
		return fmt.Errorf("load tape: %w", err)
	}

	tapeStorage, err := os.MkdirTemp("", "vhs-capture")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(tapeStorage)
	}()

	home, err := mkdirp(tapeStorage, "home")
	if err != nil {
		return err
	}
	tmp, err := mkdirp(tapeStorage, "tmp")
	if err != nil {
		return err
	}

	workDirBase := vhsTape.dirName
	if workDirBase == "" {
		workDirBase = "work"
	}
	workDir, err := mkdirp(tapeStorage, workDirBase)
	if err != nil {
		return err
	}

	var conf bytes.Buffer
	conf.WriteString("set -g prefix C-a\n")
	conf.WriteString("set -g status off\n")
	conf.WriteString("set -g default-terminal screen-256color\n")
	conf.WriteString(`set -ga terminal-overrides ",*-256color*:Tc"` + "\n")
	conf.WriteString("set -g exit-empty on\n")
	conf.WriteString("set -g exit-unattached on\n")
	fmt.Fprintf(&conf, "set -g default-shell %s\n", w.Bash)
	fmt.Fprintf(&conf, "bind-key f run-shell -b %s\n", w.TmuxFastcopy)
	conf.WriteString("set -g @fastcopy-action 'tmux load-buffer -w -'\n")

	tmuxConf := filepath.Join(home, ".tmux.conf")
	if err := os.WriteFile(tmuxConf, conf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write .tmux.conf: %w", err)
	}

	bashProfile := filepath.Join(home, ".bash_profile")
	if err := os.WriteFile(bashProfile, []byte(_bashProfile), 0o644); err != nil {
		return fmt.Errorf("write .bash_profile: %w", err)
	}

	// Environment shared by exec commands below.
	cmdout, done := PrefixedWriter(tapeCfg+": ", os.Stderr)
	defer done()
	env := []string{
		"HOME=" + home,
		"PATH=" + os.Getenv("PATH"),
		"TMUX_TMPDIR=" + tmp,
		"TERM=screen-256color",
		"PAGER=less",
		"LESS=-SR",
	}
	if v, ok := os.LookupEnv("DBUS_SESSION_BUS_ADDRESS"); ok {
		env = append(env, "DBUS_SESSION_BUS_ADDRESS="+v)
	}
	for k, v := range vhsTape.env {
		env = append(env, k+"="+v)
	}

	// Run setup script, if any.
	if sh := vhsTape.setup; sh != "" {
		cmd := exec.Command("bash", "-euo", "pipefail", "-c", sh)
		cmd.Stdout = cmdout
		cmd.Stderr = cmdout
		cmd.Env = env
		cmd.Dir = workDir
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("setup: %w", err)
		}
	}

	var tapePath string
	switch vhsTape.body.Kind {
	case tapeFile:
		tapePath = vhsTape.body.Value
	case tapeScript:
		tapePath = filepath.Join(tapeStorage, "tape")
		if err := os.WriteFile(tapePath, []byte(vhsTape.body.Value), 0o644); err != nil {
			return fmt.Errorf("write tape: %w", err)
		}
	default:
		panic("unreachable")
	}

	gif := filepath.Join(
		w.WorkDir,
		strings.TrimSuffix(filepath.Base(tapeCfg), filepath.Ext(tapeCfg))+".gif",
	) // foo.tape => foo.gif
	cmd := exec.Command("vhs", tapePath, "-o", gif)
	cmd.Stdout = cmdout
	cmd.Stderr = cmdout
	cmd.Env = env
	cmd.Dir = workDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vhs: %w", err)
	}

	return nil
}

// PrefixedWriter returns a writer that prefixes each line
// with the given prefix.
//
// Returns a function to flush remaining text.
func PrefixedWriter(prefix string, w io.Writer) (_ io.Writer, flush func()) {
	pw := &prefixWriter{writer: w, prefix: []byte(prefix)}
	return pw, func() { _ = pw.flush(false /* force */) }
}

// prefixWriter is an io.Writer that prefixes each line
// with a given string.
type prefixWriter struct {
	writer io.Writer
	prefix []byte

	// Holds buffered text for the next write or flush
	// if we haven't yet seen a newline.
	buff bytes.Buffer
}

func (w *prefixWriter) Write(bs []byte) (int, error) {
	// t.Logf adds a newline so we should not write bs as-is.
	// Instead, we'll call t.Log one line at a time.
	//
	// To handle the case when Write is called with a partial line,
	// we use a buffer.
	total := len(bs)
	for len(bs) > 0 {
		idx := bytes.IndexByte(bs, '\n')
		if idx < 0 {
			// No newline. Buffer it for later.
			w.buff.Write(bs)
			break
		}

		var line []byte
		line, bs = bs[:idx+1], bs[idx+1:]
		w.buff.Write(line)
		if err := w.flush(true /* force */); err != nil {
			return total - len(bs), err
		}
	}
	return total, nil
}

// flush flushes buffered text.
// force specifies whether to flush even if there's no newline.
func (w *prefixWriter) flush(force bool) error {
	if !force && w.buff.Len() == 0 {
		return nil
	}

	defer w.buff.Reset()
	_, err := w.writer.Write(w.prefix)
	if err == nil {
		_, err = w.writer.Write(w.buff.Bytes())
	}
	return err
}

type tapeBodyKind int

const (
	// tapeFile is a .tape file on disk.
	//
	// Value is the path to the file.
	tapeFile tapeBodyKind = iota

	// tapeScript is an inline tape script.
	//
	// Value is the script contents.
	tapeScript
)

type tapeBody struct {
	Kind  tapeBodyKind
	Value string
}

type vhsTape struct {
	body tapeBody

	dirName string // defaults to "work"
	setup   string
	env     map[string]string
}

func loadVHSTape(path string) (_ *vhsTape, err error) {
	switch ext := filepath.Ext(path); ext {
	case ".tape":
		return &vhsTape{
			body: tapeBody{
				Kind:  tapeFile,
				Value: path,
			},
		}, nil

	case ".yaml", ".yml":
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer func() {
			err = errors.Join(err, f.Close())
		}()

		return loadVHSTapeYAML(f)
	default:
		return nil, fmt.Errorf("unknown tape format: %v", ext)
	}
}

func loadVHSTapeYAML(r io.Reader) (*vhsTape, error) {
	var cfg struct {
		// Setup specifies a bash script
		// that should be run before the tape is played.
		Setup string `yaml:"setup"`

		// Tape specifices the contents of the VHS tape to run.
		Tape string `yaml:"tape"`

		// Env specifies additional environment variables
		// to set when running the setup script and the tape.
		Env map[string]string `yaml:"env"`

		// Base name of the temporary directory to use.
		//
		// Defaults to "work".
		DirName string `yaml:"dirName"`
	}

	if err := yaml.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, err
	}

	return &vhsTape{
		body: tapeBody{
			Kind:  tapeScript,
			Value: cfg.Tape,
		},
		dirName: cfg.DirName,
		setup:   cfg.Setup,
		env:     cfg.Env,
	}, nil
}

// mkdirp creates a directory and all its parents.
// It returns the path to the directory.
func mkdirp(paths ...string) (string, error) {
	path := filepath.Join(paths...)
	err := os.MkdirAll(path, 0o755)
	return path, err
}
