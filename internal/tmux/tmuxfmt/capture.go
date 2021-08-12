package tmuxfmt

import (
	"fmt"
	"strconv"
	"strings"
)

// Value receives a value from the tmux output as a string and parses it.
type Value interface {
	Set(string) error
}

type captureExpr struct {
	Expr  Expr
	Value Value
}

// Capturer captures the output of tmuxfmt expressions into Go values.
type Capturer struct {
	exprs []captureExpr
}

// Prepare prepares the specified expressions into a tmuxfmt message. The
// returned capture function will parse the resultant text and fill the
// previously recorded pointers.
func (c *Capturer) Prepare() (msg string, capure func([]byte) error) {
	exprs := c.exprs
	rendered := make([]string, len(exprs))
	for i, e := range c.exprs {
		rendered[i] = Render(e.Expr)
	}

	return strings.Join(rendered, "\t"), func(bs []byte) error {
		for i, s := range strings.Split(string(bs), "\t") {
			if i >= len(exprs) {
				break
			}

			s = strings.TrimSpace(s)
			if err := exprs[i].Value.Set(s); err != nil {
				return fmt.Errorf("capture %q: %w", rendered[i], err)
			}
		}

		return nil
	}
}

// Var records that the output of the given tmuxfmt expression should be loaded
// into the specified value.
func (c *Capturer) Var(v Value, e Expr) {
	c.exprs = append(c.exprs, captureExpr{Expr: e, Value: v})
}

// StringVar specifies that the output of the provided expression should fill
// this string pointer.
func (c *Capturer) StringVar(ptr *string, e Expr) {
	c.Var((*stringValue)(ptr), e)
}

type stringValue string

func (v *stringValue) Set(s string) error {
	*(*string)(v) = s
	return nil
}

// IntVar specifies that the output of the provided expression should be parsed
// as an integer and fill this integer pointer.
func (c *Capturer) IntVar(ptr *int, e Expr) {
	c.Var((*intValue)(ptr), e)
}

type intValue int

func (v *intValue) Set(s string) error {
	i, err := strconv.Atoi(s)
	if err == nil {
		*(*int)(v) = i
	}
	return err
}

// BoolVar specifies that the output of the provided expression should be
// parsed as a boolean and fill this boolean pointer.
func (c *Capturer) BoolVar(ptr *bool, e Expr) {
	c.Var((*boolValue)(ptr), e)
}

type boolValue bool

func (v *boolValue) Set(s string) error {
	*(*bool)(v) = len(s) > 0 && s != "0"
	return nil
}
