package tmuxopt

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"go.uber.org/multierr"
)

// Value is a receiver for a tmux option value.
type Value interface {
	Set(value string) error
}

// MapValue is a receiver for a tmux map value.
type MapValue interface {
	Put(key, value string) error
}

var _ Value = flag.Value(nil) // interface matching

// Loader loads tmux options into user-specified variables.
type Loader struct {
	Tmux tmux.Driver

	once   sync.Once
	values map[string]Value
	maps   map[string]MapValue // prefix => MapValue
}

func (l *Loader) init() {
	l.once.Do(func() {
		l.values = make(map[string]Value)
		l.maps = make(map[string]MapValue)
	})
}

// Var specifies that the given option should be loaded into the provided Value
// object.
func (l *Loader) Var(val Value, option string) {
	l.init()

	l.values[option] = val
}

// MapVar specifies that options with the given prefix should be loaded into
// the provided MapValue.
//
// To support maps, tmuxopt loader works by matching the provided prefix
// against options produced by tmux. If an option name matches the given
// prefix, the rest of that name is used as the map key and the value for that
// option as the value for that key.
//
// For example, if the prefix is, "foo-item-", then given the following
// options,
//
//	foo-item-a x
//	foo-item-b y
//	foo-item-c z
//
// We'll get the map,
//
//	{a: x, b: y, c: z}
func (l *Loader) MapVar(val MapValue, prefix string) {
	l.init()

	l.maps[prefix] = val
}

// Load loads tmux options using the underlying tmux.Driver with the provided
// request. This will fill all previously specified values and vars.
func (l *Loader) Load(req tmux.ShowOptionsRequest) (err error) {
	if len(l.values) == 0 && len(l.maps) == 0 {
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

		name, value := string(line[:idx]), line[idx+1:]

		var serr error
		if r := l.lookupValue(name); r != nil {
			serr = r.Set(readValue(value))
		} else if k, r := l.lookupMapValue(name); r != nil {
			serr = r.Put(k, readValue(value))
		} else {
			continue
		}

		if serr != nil {
			err = multierr.Append(err, fmt.Errorf("load option %q: %v", name, serr))
		}
	}

	return multierr.Append(err, scan.Err())
}

func (l *Loader) lookupValue(name string) Value {
	return l.values[name]
}

func (l *Loader) lookupMapValue(name string) (key string, v MapValue) {
	for prefix, val := range l.maps {
		if strings.HasPrefix(name, prefix) {
			return strings.TrimPrefix(name, prefix), val
		}
	}
	return name, nil
}

type stringValue string

// StringVar specifies that the given option should be loaded as a string.
func (l *Loader) StringVar(dest *string, option string) {
	l.init()

	l.Var((*stringValue)(dest), option)
}

func (v *stringValue) Set(s string) error {
	*(*string)(v) = s
	return nil
}

func readValue(v []byte) (value string) {
	if len(v) == 0 {
		return ""
	}

	value = string(v)
	// Try to unquote but don't fail if it doesn't work.
	switch value[0] {
	case '\'':
		// strconv.Unquote does not like single-quoted strings with
		// multiple characters. Invert the quotes to let
		// strconv.Unquote do the heavy-lifting and invert back.
		value = invertQuotes(value)
		defer func() {
			value = invertQuotes(value)
		}()
		fallthrough
	case '"':
		if o, err := strconv.Unquote(value); err == nil {
			value = o
		}
	}
	return value
}

var _quoteInverter = strings.NewReplacer("'", `"`, `"`, "'")

func invertQuotes(s string) string {
	return _quoteInverter.Replace(s)
}
