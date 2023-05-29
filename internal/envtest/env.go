// Package envtest provides a fake environment variable backend
// for testing purposes.
package envtest

import (
	"fmt"
)

// Empty returns an empty environment.
var Empty = Env{}

// Env represents a fake environment.
type Env struct {
	items map[string]string
}

// Pairs builds a new fake environment with the provided pairs of items. There
// must be exactly an even number of items in the list.
func Pairs(pairs ...string) (*Env, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("%d items in environment are not even", len(pairs))
	}

	m := make(map[string]string, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		k, v := pairs[i], pairs[i+1]
		m[k] = v
	}
	return &Env{m}, nil
}

// MustPairs builds an Env with the provided items, panicking if it fails.
func MustPairs(items ...string) *Env {
	e, err := Pairs(items...)
	if err != nil {
		panic(err)
	}
	return e
}

// Getenv is an analog for the os.Getenv operation.
func (e *Env) Getenv(k string) string {
	if e == nil {
		return ""
	}

	return e.items[k]
}
