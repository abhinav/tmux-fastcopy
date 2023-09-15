package fastcopy

import (
	"testing"

	tcell "github.com/gdamore/tcell/v2"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc   string
		give   Range
		length int
		str    string
	}{
		{desc: "zero", str: "[0, 0)"},
		{
			desc:   "one",
			give:   Range{0, 5},
			length: 5,
			str:    "[0, 5)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.length, tt.give.Len(), "length")
			assert.Equal(t, tt.str, tt.give.String(), "string")
		})
	}
}

func sampleStyle() Style {
	return Style{
		Normal:         tcell.StyleDefault,
		Match:          tcell.StyleDefault.Foreground(tcell.ColorGreen),
		SkippedMatch:   tcell.StyleDefault.Foreground(tcell.ColorGray),
		HintLabel:      tcell.StyleDefault.Foreground(tcell.ColorRed),
		HintLabelInput: tcell.StyleDefault.Foreground(tcell.ColorYellow),
	}
}

//nolint:paralleltest // shared state between subtests
func TestWidget(t *testing.T) {
	t.Parallel()

	mockCtrl := gomock.NewController(t)
	handler := NewMockHandler(mockCtrl)
	style := sampleStyle()

	//      0 1 2   3
	//    [(f o)o ] \n
	//  4 [ b(a r)] \n
	//  8 [ b(a z)] \n
	// 12 [(q u)x ] \n
	w := (&WidgetConfig{
		Text: "foo\nbar\nbaz\nqux",
		Matches: []Match{
			{"p", Range{0, 2}},   // (fo)
			{"q", Range{5, 7}},   // (ar)
			{"r", Range{9, 11}},  // (az)
			{"s", Range{12, 14}}, // (qu)
		},
		HintAlphabet: []rune("ab"),
		Handler:      handler,
		Style:        style,
		generateHints: func([]rune, string, []Match) []hint {
			return []hint{
				{Label: "aa", Text: "fo", Matches: []Match{{"p", Range{0, 2}}}},   // (fo)
				{Label: "bb", Text: "ar", Matches: []Match{{"q", Range{5, 7}}}},   // (ar)
				{Label: "ba", Text: "az", Matches: []Match{{"r", Range{9, 11}}}},  // (az)
				{Label: "ab", Text: "qu", Matches: []Match{{"p", Range{12, 14}}}}, // (qu)
			}
		},
	}).Build()

	screen := tcell.NewSimulationScreen("")
	screen.SetSize(3, 3)
	screen.Clear()
	w.Draw(screen)

	t.Run("mouse event", func(t *testing.T) {
		ev := tcell.NewEventMouse(1, 1, tcell.Button1, 0)
		assert.False(t, w.HandleEvent(ev),
			"widget cannot handle mouse events yet")
	})

	t.Run("partial input", func(t *testing.T) {
		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0)),
			"widget must handle key event")

		assert.Equal(t, "a", w.Input())

		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyBackspace, 0, 0)),
			"widget must handle backspace event")

		assert.Empty(t, w.Input())
	})

	t.Run("select", func(t *testing.T) {
		handler.EXPECT().
			HandleSelection(Selection{Text: "az", Matchers: []string{"r"}})

		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0)))
		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0)))
	})

	t.Run("shift select", func(t *testing.T) {
		handler.EXPECT().
			HandleSelection(Selection{
				Text:     "qu",
				Matchers: []string{"p"},
				Shift:    true,
			})

		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0)))
		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'B', 0)))
	})

	t.Run("multi-select", func(t *testing.T) {
		handler.EXPECT().
			HandleSelection(Selection{
				Text:     "fo ar",
				Matchers: []string{"p", "q"},
			})

		// enter multi-select mode
		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyTab, 0, 0)))

		// Select one.
		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0)))
		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'a', 0)))

		// Select another.
		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0)))
		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyRune, 'b', 0)))

		// Accept selection.
		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyTab, 0, 0)))
	})

	t.Run("multi-select no match", func(t *testing.T) {
		// Enter multi-select and accept right away.
		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyTab, 0, 0)))
		assert.True(t,
			w.HandleEvent(tcell.NewEventKey(tcell.KeyEnter, 0, 0)))
	})
}
