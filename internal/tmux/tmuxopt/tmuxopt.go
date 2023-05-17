// Package tmuxopt provides an API for loading and parsing tmux options
// into Go variables.
//
// It provides an API similar to the flag package, but for tmux options.
package tmuxopt

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"go.uber.org/multierr"
)

// Transformer TODO
type Transformer func(k, v string) (outk, outv string, err error)

// Loader loads tmux options into user-specified variables.
type Loader struct {
	Tmux   tmux.Driver
	Prefix string

	once               sync.Once
	prefix             []byte
	prefixTransformers map[string][]Transformer // prefix => transformers
}

func (l *Loader) init() {
	l.once.Do(func() {
		l.prefix = []byte(l.Prefix)
		l.prefixTransformers = make(map[string][]Transformer)
	})
}

// Transformer object.
func (l *Loader) PrefixTransformer(prefix string, tr Transformer) {
	l.init()

	l.prefixTransformers[prefix] = append(l.prefixTransformers[prefix], tr)
}

var _space = []byte{' '}

// Load loads tmux options using the underlying tmux.Driver with the provided
// request. This will fill all previously specified values and vars.
func (l *Loader) Load(req tmux.ShowOptionsRequest) (err error) {
	out, err := l.Tmux.ShowOptions(req)
	if err != nil {
		return err
	}

	scan := bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		keyb, valb, ok := bytes.Cut(scan.Bytes(), _space)
		if !ok {
			continue
		}

		keyb, ok = bytes.CutPrefix(keyb, l.prefix)
		if !ok {
			continue
		}

		key := string(keyb)
		val := readValue(valb)
		for _, t := range l.prefixTransformers[key] {
			nk, nv, err := t(key, val)
			if err != nil {
				return fmt.Errorf("transform %q: %w", key, err)
			}
			key, val = nk, nv
		}
	}

	return multierr.Append(err, scan.Err())
}

// type stringValue string
//
// // StringVar specifies that the given option should be loaded as a string.
// func (l *Loader) StringVar(dest *string, option string) {
// 	l.init()
//
// 	l.Var((*stringValue)(dest), option)
// }
//
// func (v *stringValue) Set(s string) error {
// 	*(*string)(v) = s
// 	return nil
// }
//
// type boolValue bool
//
// // BoolVar specifies that the given option should be loaded as a boolean.
// func (l *Loader) BoolVar(dest *bool, option string) {
// 	l.init()
//
// 	l.Var((*boolValue)(dest), option)
// }
//
// func (v *boolValue) Set(s string) error {
// 	switch strings.ToLower(strings.TrimSpace(s)) {
// 	case "on", "yes", "true", "1":
// 		*(*bool)(v) = true
// 	case "off", "no", "false", "0":
// 		*(*bool)(v) = false
// 	default:
// 		return fmt.Errorf("invalid boolean value %q", s)
// 	}
// 	return nil
// }

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
