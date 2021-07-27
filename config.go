package main

import (
	"flag"
	"io"
	"os"
)

type config struct {
	Pane    string
	Verbose bool
	LogFile string
	Action  string
}

func newConfig(flag *flag.FlagSet) *config {
	var c config

	// No help here because we put it all in _usage.
	flag.StringVar(&c.Pane, "pane", "", "")
	flag.StringVar(&c.Action, "action", "", "")
	flag.BoolVar(&c.Verbose, "verbose", false, "")
	flag.StringVar(&c.LogFile, "log", "", "")

	return &c
}

// Args rebuilds a list of arguments from which this configuration may be
// parsed.
func (c *config) Args() []string {
	var args []string
	if len(c.Pane) > 0 {
		args = append(args, "-pane", c.Pane)
	}
	if c.Verbose {
		args = append(args, "-verbose")
	}
	if len(c.LogFile) > 0 {
		args = append(args, "-log", c.LogFile)
	}
	if len(c.Action) > 0 {
		args = append(args, "-action", c.Action)
	}
	return args
}

// BuildLogWriter builds an io.Writer based on the configuration. It may be
// called any number of times and will return the same values.
func (c *config) BuildLogWriter(stderr io.Writer) (w io.Writer, close func(), err error) {
	if len(c.LogFile) == 0 {
		return stderr, func() {}, nil
	}

	f, err := os.OpenFile(c.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	return f, func() { f.Close() }, nil
}
