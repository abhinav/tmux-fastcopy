package log

import "log/slog"

// OmitEmpty builds an attribute using the given constructor function,
// but if the value is the zero value for its type,
// it skips the attribute.
func OmitEmpty[T comparable](fn func(string, T) slog.Attr, name string, value T) slog.Attr {
	var zero T
	if value == zero {
		return slog.Attr{} // ignore
	}
	return fn(name, value)
}
