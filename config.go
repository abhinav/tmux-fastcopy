package main

import (
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/abhinav/tmux-fastcopy/internal/must"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxopt"
)

var _defaultRegexes = map[string]string{
	"ipv4":     `\b\d{1,3}(?:\.\d{1,3}){3}\b`,
	"gitsha":   `\b[0-9a-f]{7,40}\b`,
	"hexaddr":  `\b(?i)0x[0-9a-f]{2,}\b`,
	"hexcolor": `(?i)#(?:[0-9a-f]{3}|[0-9a-f]{6})\b`,
	"int":      `(?:-?|\b)\d{4,}\b`,
	"path":     `(?:[^\w\-\.~/]|\A)(([\w\-\.]+|~)?(/[\w\-\.]+){2,})\b`,
	"uuid":     `\b(?i)[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}\b`,
	"isodate":  `\d{4}-\d{2}-\d{2}`,
}

// regexes is a map from regex name to body. If body is empty, this regex
// should be skipped.
type regexes map[string]string

func (m *regexes) Put(k, v string) error {
	if len(k) == 0 {
		return errors.New("regex must have a name")
	}

	if *m == nil {
		*m = make(map[string]string)
	}
	(*m)[k] = v
	return nil
}

func (m regexes) Flags() (args []string) {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		args = append(args, "-regex", name+":"+m[name])
	}

	return args
}

func (m regexes) String() string {
	return fmt.Sprint(m.Flags())
}

func (m *regexes) Set(v string) error {
	idx := strings.IndexByte(v, ':')
	if idx < 0 {
		return errors.New("regex flags must be in the form NAME:REGEX")
	}

	return m.Put(v[:idx], v[idx+1:])
}

func (m *regexes) FillFrom(o regexes) {
	for k, v := range o {
		if _, ok := (*m)[k]; !ok {
			err := m.Put(k, v)
			must.NotErrorf(err, "unexpected invalid key %q", k)
		}
	}
}

type config struct {
	Pane        string
	Action      string
	ShiftAction string
	Alphabet    alphabet
	Verbose     bool
	Regexes     regexes
	Tmux        string
	LogFile     string
}

// Generates a new default configuration.
func defaultConfig(cfg *config) *config {
	return &config{
		Action:   fmt.Sprintf("%v load-buffer -", cfg.Tmux),
		Alphabet: _defaultAlphabet,
		Regexes:  _defaultRegexes,
	}
}

func (c *config) RegisterFlags(flag *flag.FlagSet) {
	// No help here because we put it all in _usage.
	flag.StringVar(&c.Pane, "pane", "", "")
	flag.StringVar(&c.Action, "action", "", "")
	flag.StringVar(&c.ShiftAction, "shift-action", "", "")
	flag.Var(&c.Alphabet, "alphabet", "")
	flag.Var(&c.Regexes, "regex", "")
	flag.BoolVar(&c.Verbose, "verbose", false, "")
	flag.StringVar(&c.LogFile, "log", "", "")
	flag.StringVar(&c.Tmux, "tmux", "tmux", "")
}

func (c *config) RegisterOptions(load *tmuxopt.Loader) {
	load.StringVar(&c.Action, "@fastcopy-action")
	load.StringVar(&c.ShiftAction, "@fastcopy-shift-action")
	load.Var(&c.Alphabet, "@fastcopy-alphabet")
	load.MapVar(&c.Regexes, "@fastcopy-regex-")
}

// FillFrom updates this config object, filling empty values with values from
// the provided struct but not overwriting those that are already set.
func (c *config) FillFrom(o *config) {
	if len(c.Pane) == 0 {
		c.Pane = o.Pane
	}
	if len(c.Action) == 0 {
		c.Action = o.Action
	}
	if len(c.ShiftAction) == 0 {
		c.ShiftAction = o.ShiftAction
	}
	if len(c.Alphabet) == 0 {
		c.Alphabet = o.Alphabet
	}
	if len(c.LogFile) == 0 {
		c.LogFile = o.LogFile
	}
	if len(c.Tmux) == 0 {
		c.Tmux = o.Tmux
	}
	c.Regexes.FillFrom(o.Regexes)
	c.Verbose = c.Verbose || o.Verbose
}

// Flags rebuilds a list of arguments from which this configuration may be
// parsed.
func (c *config) Flags() []string {
	var args []string
	if len(c.Pane) > 0 {
		args = append(args, "-pane", c.Pane)
	}
	if len(c.Action) > 0 {
		args = append(args, "-action", c.Action)
	}
	if len(c.ShiftAction) > 0 {
		args = append(args, "-shift-action", c.ShiftAction)
	}
	if len(c.Alphabet) > 0 {
		args = append(args, "-alphabet", c.Alphabet.String())
	}
	args = append(args, c.Regexes.Flags()...)
	if c.Verbose {
		args = append(args, "-verbose")
	}
	if len(c.LogFile) > 0 {
		args = append(args, "-log", c.LogFile)
	}
	if len(c.Tmux) > 0 {
		args = append(args, "-tmux", c.Tmux)
	}
	return args
}
