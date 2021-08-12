package tmuxopt

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"strconv"
	"sync"

	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"go.uber.org/multierr"
)

// Value is a receiver for a tmux option value.
type Value interface {
	Set(value string) error
}

var _ Value = flag.Value(nil) // interface matching

// Loader loads tmux options inot user-specified variables.
type Loader struct {
	Tmux tmux.Driver

	once   sync.Once
	values map[string]Value
}

func (l *Loader) init() {
	l.once.Do(func() { l.values = make(map[string]Value) })
}

// Var specifies that the given option should be loaded into the provided Value
// object.
func (l *Loader) Var(val Value, option string) {
	l.init()

	l.values[option] = val
}

// StringVar specifies that the given option should be loaded as a string.
func (l *Loader) StringVar(dest *string, option string) {
	l.init()

	l.Var((*stringValue)(dest), option)
}

// Load loads tmux options using the underlying tmux.Driver with the provided
// request. This will fill all previously specified values and vars.
func (l *Loader) Load(req tmux.ShowOptionsRequest) (err error) {
	if len(l.values) == 0 {
		return nil
	}

	out, err := l.Tmux.ShowOptions(req)
	if err != nil {
		return err
	}

	scan := bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		line := scan.Bytes()

		idx := bytes.IndexByte(line, ' ')
		if idx < 0 {
			continue
		}

		name := line[:idx]
		r, ok := l.values[string(name)]
		if !ok {
			continue
		}

		value := string(line[idx+1:])
		if len(value) > 0 {
			// Try to unquote but don't fail if it doesn't work.
			switch value[0] {
			case '"', '\'':
				if o, err := strconv.Unquote(value); err == nil {
					value = o
				}
			}
		}

		if serr := r.Set(value); serr != nil {
			err = multierr.Append(err, fmt.Errorf("load option %q: %v", name, serr))
		}
	}

	return multierr.Append(err, scan.Err())
}

type stringValue string

func (v *stringValue) Set(s string) error {
	if len(s) > 0 {
		// Try to unquote but don't fail if it doesn't work.
		switch s[0] {
		case '"', '\'':
			o, err := strconv.Unquote(s)
			if err == nil {
				s = o
			}
		}
	}

	*(*string)(v) = s
	return nil
}
