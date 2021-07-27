package tmuxfmt

import (
	"fmt"
	"strconv"
	"strings"
)

type captureExpr struct {
	Expr    Expr
	Capture func(string) error
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
			if err := exprs[i].Capture(s); err != nil {
				return fmt.Errorf("capture %q: %w", rendered[i], err)
			}
		}

		return nil
	}
}

// StringVar specifies that the output of the provided expression should fill
// this string pointer.
func (c *Capturer) StringVar(ptr *string, e Expr) {
	c.exprs = append(c.exprs, captureExpr{
		Expr: e,
		Capture: func(v string) error {
			*ptr = v
			return nil
		},
	})
}

// IntVar specifies that the output of the provided expression should be parsed
// as an integer and fill this integer pointer.
func (c *Capturer) IntVar(ptr *int, e Expr) {
	c.exprs = append(c.exprs, captureExpr{
		Expr: e,
		Capture: func(v string) error {
			i, err := strconv.Atoi(v)
			*ptr = i
			return err
		},
	})
}

// BoolVar specifies that the output of the provided expression should be
// parsed as a boolean and fill this boolean pointer.
func (c *Capturer) BoolVar(ptr *bool, e Expr) {
	c.exprs = append(c.exprs, captureExpr{
		Expr: e,
		Capture: func(v string) error {
			*ptr = len(v) > 0 && v != "0"
			return nil
		},
	})
}
