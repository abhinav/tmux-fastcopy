package tmuxfmt

import (
	"bytes"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Render renders the provided tmux expressions separated by the given
// delimiter in a format compatible with tmux's FORMATS section.
func Render(e Expr) string {
	var out strings.Builder
	render(&out, e, false)
	return out.String()
}

func render(w *strings.Builder, e Expr, escapeString bool) {
	switch e := e.(type) {
	case String:
		if escapeString {
			renderStringEscaped(w, []byte(e))
		} else {
			w.WriteString(string(e))
		}

	case Int:
		w.WriteString(strconv.Itoa(int(e)))

	case Var:
		w.WriteString("#{")
		w.WriteString(string(e))
		w.WriteString("}")

	case Ternary:
		w.WriteString("#{?")
		render(w, e.Cond, true)
		w.WriteString(",")
		render(w, e.Then, true)
		w.WriteString(",")
		render(w, e.Else, true)
		w.WriteString("}")

	case Binary:
		w.WriteString("#{")
		w.WriteString(e.Op.String())
		w.WriteString(":")
		render(w, e.LHS, true)
		w.WriteString(",")
		render(w, e.RHS, true)
		w.WriteString("}")
	}
}

const _escapedRunes = ",#}"

func renderStringEscaped(w *strings.Builder, b []byte) {
	for len(b) > 0 {
		idx := bytes.IndexAny(b, _escapedRunes)
		if idx < 0 {
			w.Write(b)
			return
		}

		w.Write(b[:idx])
		b = b[idx:]

		r, sz := utf8.DecodeRune(b)
		w.WriteRune('#')
		w.WriteRune(r)
		b = b[sz:]
	}
}
