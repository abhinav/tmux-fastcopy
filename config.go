package main

import (
	"flag"

	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxopt"
)

var _defaultConfig = config{
	Action:   _defaultAction,
	Alphabet: _defaultAlphabet,
}

type config struct {
	Pane     string
	Action   string
	Alphabet alphabet
	Verbose  bool
}

func (c *config) RegisterFlags(flag *flag.FlagSet) {
	// No help here because we put it all in _usage.
	flag.StringVar(&c.Pane, "pane", "", "")
	flag.StringVar(&c.Action, "action", "", "")
	flag.Var(&c.Alphabet, "alphabet", "")
	flag.BoolVar(&c.Verbose, "verbose", false, "")
}

func (c *config) RegisterOptions(load *tmuxopt.Loader) {
	load.StringVar(&c.Action, "@fastcopy-action")
	load.Var(&c.Alphabet, "@fastcopy-alphabet")
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
	if c.Verbose {
		args = append(args, "-verbose")
	}
	return args
}
