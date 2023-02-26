package tmuxfmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapturer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc    string
		give    []byte        // output from tmux
		exprs   []Expr        // expressions to prepare
		want    []interface{} // expected values in-order
		wantErr string        // error (if set, want is used only to get types)
	}{
		{
			desc:  "string",
			exprs: []Expr{Var("pane_id")},
			give:  []byte("%42\n"),
			want:  []interface{}{"%42"},
		},
		{
			desc:  "int",
			exprs: []Expr{Var("height")},
			give:  []byte("42"),
			want:  []interface{}{42},
		},
		{
			desc:  "bool",
			exprs: []Expr{Var("window_zoomed")},
			give:  []byte("1\n"),
			want:  []interface{}{true},
		},
		{
			desc: "multiple",
			exprs: []Expr{
				Var("pane_id"),
				Var("height"),
				Var("window_zoomed"),
			},
			give: []byte("%42	100	true\n"),
			want: []interface{}{"%42", 100, true},
		},
		{
			desc: "empty",
			exprs: []Expr{
				Var("pane_in_state"),
				Var("pane_state"),
			},
			give: []byte("0	\n"),
			want: []interface{}{false, ""},
		},
		{
			desc:    "int/error",
			exprs:   []Expr{Var("height")},
			give:    []byte("four\n"),
			want:    []interface{}{0},
			wantErr: `capture "#{height}": .*invalid syntax`,
		},
		{
			desc:  "too many results",
			exprs: []Expr{Var("pane_width")},
			give:  []byte("80	40	10\n"),
			want:  []interface{}{80},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			require.Len(t, tt.want, len(tt.exprs), "invalid test: "+
				"number of expressions must match the"+
				"number of expected values "+
				"if an error is not expectedd")

			got := make([]interface{}, len(tt.exprs))  // list of pointers
			want := make([]interface{}, len(tt.exprs)) // list of pointers

			var c Capturer
			for i, expr := range tt.exprs {
				switch w := tt.want[i].(type) {
				case string:
					g := new(string)
					c.StringVar(g, expr)
					want[i] = &w
					got[i] = g

				case int:
					g := new(int)
					c.IntVar(g, expr)
					want[i] = &w
					got[i] = g

				case bool:
					g := new(bool)
					c.BoolVar(g, expr)
					want[i] = &w
					got[i] = g

				default:
					t.Fatalf("unsupported want: %v (%T)", w, w)
				}
			}

			_, capture := c.Prepare()
			err := capture(tt.give)
			if len(tt.wantErr) > 0 {
				assert.Error(t, err)
				assert.Regexp(t, tt.wantErr, err.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}
