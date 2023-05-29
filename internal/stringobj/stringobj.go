// Package stringobj aids in writing String methods for objects
// with a JSON-like output.
package stringobj

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Builder helps build String functions for objects that skip zero-value
// attributes.
type Builder struct {
	attrs []string
}

// Put adds the given attribute-value pair to the builder, skipping it if the
// value is a zero value.
func (b *Builder) Put(name string, value interface{}) {
	// This whole module is icky; we can do something like
	// zap.ObjectEncoders later.
	if value == nil {
		return
	}
	if v := reflect.ValueOf(value); v.IsZero() {
		return
	}
	b.attrs = append(b.attrs, fmt.Sprintf("%s: %v", name, value))
}

// String returns the final string representation.
func (b *Builder) String() string {
	sort.Strings(b.attrs)

	var out strings.Builder
	out.WriteRune('{')
	for i, attr := range b.attrs {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(attr)
	}
	out.WriteRune('}')
	return out.String()
}
