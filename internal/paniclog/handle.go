// Package paniclog provides a handler for panicking code
// that logs the panic to an io.Writer.
package paniclog

import (
	"errors"
	"fmt"
	"io"
	"runtime/debug"
)

// Handle handles a panic value, logging it to the given io.Writer. Returns the
// error version of the panic, if any.
func Handle(pval interface{}, w io.Writer) error {
	if pval == nil {
		return nil
	}

	fmt.Fprintf(w, "panic: %v\n%s", pval, debug.Stack())

	var err error
	switch pval := pval.(type) {
	case string:
		err = errors.New(pval)
	case error:
		err = pval
	default:
		err = fmt.Errorf("panic: %v", pval)
	}
	return err
}

// Recover recovers a panic and appends it into the given error pointer.
func Recover(err *error, w io.Writer) {
	if pval := recover(); pval != nil {
		*err = Handle(pval, w)
	}
}
