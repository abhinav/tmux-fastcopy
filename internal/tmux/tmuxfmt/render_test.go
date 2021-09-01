package tmuxfmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRender(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give Expr
		want string
	}{
		{
			desc: "string",
			give: String("foo"),
			want: "foo",
		},
		{
			desc: "int",
			give: Int(42),
			want: "42",
		},
		{
			desc: "var",
			give: Var("pane_id"),
			want: "#{pane_id}",
		},
		{
			desc: "ternary",
			give: Ternary{
				Cond: Var("pane_in_mode"),
				Then: Var("pane_mode"),
				Else: String("normal-mode"),
			},
			want: "#{?#{pane_in_mode},#{pane_mode},normal-mode}",
		},
		{
			desc: "ternary/string escape",
			give: Ternary{
				Cond: Var("pane_in_mode"),
				Then: String("a,b"),
				Else: String("x,y"),
			},
			want: "#{?#{pane_in_mode},a#,b,x#,y}",
		},
		{
			desc: "binary/eq",
			give: Binary{
				Op:  Equals,
				LHS: Var("cursor_x"),
				RHS: Var("copy_cursor_x"),
			},
			want: "#{==:#{cursor_x},#{copy_cursor_x}}",
		},
		{
			desc: "binary/ne",
			give: Binary{
				Op:  NotEquals,
				LHS: Var("cursor_x"),
				RHS: Var("cursor_y"),
			},
			want: "#{!=:#{cursor_x},#{cursor_y}}",
		},
		{
			desc: "binary/lt",
			give: Binary{
				Op:  LessThan,
				LHS: Var("cursor_x"),
				RHS: Int(42),
			},
			want: "#{<:#{cursor_x},42}",
		},
		{
			desc: "binary/gt",
			give: Binary{
				Op:  GreaterThan,
				LHS: Var("cursor_x"),
				RHS: Var("scroll_position"),
			},
			want: "#{>:#{cursor_x},#{scroll_position}}",
		},
		{
			desc: "binary/lte",
			give: Binary{
				Op:  LessThanEquals,
				LHS: Var("cursor_x"),
				RHS: Var("pane_width"),
			},
			want: "#{<=:#{cursor_x},#{pane_width}}",
		},
		{
			desc: "binary/gte",
			give: Binary{
				Op:  GreaterThanEquals,
				LHS: Var("cursor_x"),
				RHS: Int(0),
			},
			want: "#{>=:#{cursor_x},0}",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, Render(tt.give))
		})
	}
}
