package main

import (
	"errors"
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxopt"
)

var _defaultRegexes = map[string]string{
	"ipv4":     `\b\d{1,3}(?:\.\d{1,3}){3}\b`,
	"gitsha":   `\b[0-9a-f]{7,40}\b`,
	"hexaddr":  `\b(?i)0x[0-9a-f]{2,}\b`,
	"hexcolor": `(?i)#(?:[0-9a-f]{3}|[0-9a-f]{6})\b`,
	"int":      `(?:-?|\b)\d{4,}\b`,
	"path":     `(?:[\w\-\.]+|~)?(?:/[\w\-\.]+){2,}\b`,
	"uuid":     `\b(?i)[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}\b`,
	"isodate":  `\d{4}-\d{2}-\d{2}`,
}

var _defaultConfig = config{
	Action:   _defaultAction,
	Alphabet: _defaultAlphabet,
	Regexes:  _defaultRegexes,
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

	if err := m.Put(v[:idx], v[idx+1:]); err != nil {
		return err
	}

	return nil
}

func (m *regexes) FillFrom(o regexes) {
	for k, v := range o {
		if _, ok := (*m)[k]; !ok {
			m.Put(k, v)
		}
	}
}

type config struct {
	Pane     string
	Action   string
	Alphabet alphabet
	Verbose  bool
	Regexes  regexes
}

func (c *config) RegisterFlags(flag *flag.FlagSet) {
	// No help here because we put it all in _usage.
	flag.StringVar(&c.Pane, "pane", "", "")
	flag.StringVar(&c.Action, "action", "", "")
	flag.Var(&c.Alphabet, "alphabet", "")
	flag.Var(&c.Regexes, "regex", "")
	flag.BoolVar(&c.Verbose, "verbose", false, "")
}

func (c *config) RegisterOptions(load *tmuxopt.Loader) {
	load.StringVar(&c.Action, "@fastcopy-action")
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
	if len(c.Alphabet) == 0 {
		c.Alphabet = o.Alphabet
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
	if len(c.Alphabet) > 0 {
		args = append(args, "-alphabet", c.Alphabet.String())
	}
	args = append(args, c.Regexes.Flags()...)
	if c.Verbose {
		args = append(args, "-verbose")
	}
	return args
}
