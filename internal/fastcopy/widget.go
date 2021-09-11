// Package fastcopy implements the core fastcopy functionality.
package fastcopy

import (
	"fmt"
	"sync"

	"github.com/abhinav/tmux-fastcopy/internal/ui"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

// Range specifies a range of offsets in a text, referring to the [start:end)
// subslice of the text.
type Range struct{ Start, End int }

func (r Range) String() string {
	return fmt.Sprintf("[%v, %v)", r.Start, r.End)
}

// Len reports the length of this range.
func (r *Range) Len() int {
	return r.End - r.Start
}

// Style configures the display style of the widget.
type Style struct {
	Normal       tcell.Style // normal text
	Match        tcell.Style // matched text
	SkippedMatch tcell.Style // matched text that is not selected

	HintLabel      tcell.Style // labels for hints
	HintLabelInput tcell.Style // typed portion of hints
}

// Handler handles events from the widget.
type Handler interface {
	// HandleSelection reports the hint label and the corresponding matched
	// text.
	HandleSelection(hintLabel, text string)
}

// WidgetConfig configures the fastcopy widget.
type WidgetConfig struct {
	// Text to display on the widget.
	Text string

	// Matched offsets in text.
	Matches []Range

	// Alphabet we'll use to generate labels.
	HintAlphabet []rune

	// Handler handles events from the widget. This includes hint
	// selection.
	Handler Handler

	Style Style
}

// Widget is the main fastcopy widget. It displays some fixed text with zero or
// more hints and unique prefix-free labels next to each hint to select that
// label.
type Widget struct {
	style   Style
	handler Handler
	textw   *ui.AnnotatedText

	hints        []hint
	hintsByLabel map[string]int // label -> hints[i]

	// Mutable attributes:

	mu    sync.RWMutex
	input string // text input so far
}

// Build builds a new Fastcopy widget using the provided configuration.
func (cfg *WidgetConfig) Build() *Widget {
	hints := generateHints(cfg.HintAlphabet, cfg.Text, cfg.Matches)
	byLabel := make(map[string]int, len(hints))

	for i, hint := range hints {
		byLabel[hint.Label] = i
	}

	w := &Widget{
		textw: &ui.AnnotatedText{
			Text:  cfg.Text,
			Style: cfg.Style.Normal,
		},
		style:        cfg.Style,
		handler:      cfg.Handler,
		hints:        hints,
		hintsByLabel: byLabel,
	}
	w.annotateText()
	return w
}

// Draw draws the widget onto the provided view.
func (w *Widget) Draw(view views.View) {
	w.textw.Draw(view)
}

// HandleEvent handles input for the widget. This only responds to text input,
// and delegates everything else to the caller.
func (w *Widget) HandleEvent(ev tcell.Event) (handled bool) {
	ek, ok := ev.(*tcell.EventKey)
	if !ok {
		return false
	}

	switch ek.Key() {
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		handled = true
		w.mu.Lock()
		if n := len(w.input); n > 0 {
			w.input = w.input[:n-1]
			defer w.inputChanged()
		}
		w.mu.Unlock()

	case tcell.KeyRune:
		handled = true
		w.mu.Lock()
		w.input += string(ek.Rune())
		defer w.inputChanged()
		w.mu.Unlock()
	}

	return handled
}

func (w *Widget) inputChanged() {
	// We use prefix-free hint labels, so if a label matches the input
	// exactly, we have a guarantee that this is a match.
	defer w.annotateText()

	var h hint

	w.mu.RLock()
	idx, ok := w.hintsByLabel[w.input]
	if ok {
		h = w.hints[idx]
	}
	w.mu.RUnlock()

	if ok && w.handler != nil {
		w.handler.HandleSelection(h.Label, h.Text)
	}
}

func (w *Widget) annotateText() {
	w.mu.Lock()
	defer w.mu.Unlock()

	var anns []ui.TextAnnotation
	for _, hint := range w.hints {
		anns = append(anns, hint.Annotations(w.input, w.style)...)
	}

	w.textw.SetAnnotations(anns...)
}
