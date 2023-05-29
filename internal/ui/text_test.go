package ui

import (
	"testing"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/stretchr/testify/assert"
)

func TestDrawText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		text string
		w, h int // defaults to 10
		give Pos
		want Pos
	}{
		{
			desc: "empty",
			give: Pos{1, 2},
			want: Pos{1, 2},
		},
		{
			desc: "single line",
			text: "hello",
			want: Pos{5, 0},
		},
		{
			desc: "multi line",
			text: "hello\nworld",
			want: Pos{5, 1},
		},
		{
			desc: "multi line/end with newline",
			text: "hello\nworld\n",
			want: Pos{0, 2},
		},
		{
			desc: "out of bounds/x",
			w:    4,
			text: "hello",
			want: Pos{1, 1},
		},
		{
			desc: "out of bounds/y",
			h:    2,
			text: "h\ne\nl\nl\no",
			want: Pos{0, 2},
		},
		{
			desc: "wide char",
			text: "世",
			want: Pos{2, 0},
		},
		{
			desc: "zero width char",
			text: "a\x00b",
			want: Pos{2, 0},
		},
		{
			desc: "combining rune",
			text: string([]rune{0x1f3f3, 0xfe0f, 0x200d, 0x1f308}), // 🏳️‍🌈
			want: Pos{1, 0},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			w, h := 10, 10
			if tt.w > 0 {
				w = tt.w
			}
			if tt.h > 0 {
				h = tt.h
			}
			scr := NewTestScreen(t, w, h)

			got := DrawText(
				tt.text, tcell.StyleDefault, scr, tt.give,
			)
			assert.Equal(t, tt.want, got)
		})
	}
}
