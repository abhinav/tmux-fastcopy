package stub

// Replace replaces the given value for the duration of the test.
func Replace[V any](dst *V, val V) (restore func()) {
	old := *dst
	*dst = val
	return func() {
		*dst = old
	}
}
