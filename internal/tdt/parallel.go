package tdt

import "github.com/maxatome/go-testdeep/td"

// Parallel marks the given td.T test as parallel.
func Parallel(t *td.T) {
	p, ok := t.TB.(interface{ Parallel() })
	if ok {
		p.Parallel()
	}

	// TODO: Delete once https://github.com/maxatome/go-testdeep/pull/150
	// is released.
}
