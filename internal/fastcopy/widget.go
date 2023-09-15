// Package fastcopy implements the core fastcopy functionality.
package fastcopy

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/abhinav/tmux-fastcopy/internal/ui"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

// Match is a single entry matched by fastcopy.
type Match struct {
	// Matcher is the name of the matcher that found this match.
	Matcher string

	// Range identifies the matched area.
	Range Range
}

func (m Match) String() string {
	return fmt.Sprintf("(%q) %v", m.Matcher, m.Range)
}

// Range specifies a range of offsets in a text, referring to the [start:end)
// subslice of the text.
type Range struct{ Start, End int }

func (r Range) String() string {
	return fmt.Sprintf("[%v, %v)", r.Start, r.End)
}

// Len reports the length of this range.
func (r Range) Len() int {
	return r.End - r.Start
}

// Style configures the display style of the widget.
type Style struct {
	Normal       tcell.Style // normal text
	Match        tcell.Style // matched text
	SkippedMatch tcell.Style // matched text that is not selected

	HintLabel      tcell.Style // labels for hints
	HintLabelInput tcell.Style // typed portion of hints

	// Multi-select mode:
	SelectedMatch tcell.Style // one of the selected matches
	DeselectLabel tcell.Style // label for deselection
}

// Selection is a choice made by the user in the fastcopy UI.
type Selection struct {
	// Text is the matched text.
	Text string

	// Matchers is a list of names of matchers that matched this text.
	// Invariant: this list contains at least one item.
	Matchers []string

	// Shift reports whether shift was pressed when this value was
	// selected.
	Shift bool
}

// Handler handles events from the widget.
type Handler interface {
	// HandleSelection reports the hint label and the corresponding matched
	// text.
	HandleSelection(Selection)
}

//go:generate mockgen -destination mock_handler_test.go -package fastcopy github.com/abhinav/tmux-fastcopy/internal/fastcopy Handler

// WidgetConfig configures the fastcopy widget.
type WidgetConfig struct {
	// Text to display on the widget.
	Text string

	// Matched offsets in text.
	Matches []Match

	// Alphabet we'll use to generate labels.
	HintAlphabet []rune

	// Handler handles events from the widget. This includes hint
	// selection.
	Handler Handler

	// Style configures the look of the widget.
	Style Style

	// Internal override for generateHints.
	generateHints func([]rune, string, []Match) []hint
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

	mu          sync.RWMutex
	input       string // text input so far
	shiftDown   bool   // whether shift was pressed
	multiSelect bool   // whether in multi select mode
}

// Build builds a new Fastcopy widget using the provided configuration.
func (cfg *WidgetConfig) Build() *Widget {
	generateHints := generateHints
	if cfg.generateHints != nil {
		generateHints = cfg.generateHints
	}

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

// Input reports the text input into the label so far to partially select a
// label.
func (w *Widget) Input() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.input
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

	case tcell.KeyTab:
		handled = true
		if !w.multiSelect {
			w.multiSelect = true
		} else {
			w.multiSelect = false
			w.handleSelection()
		}

	case tcell.KeyEnter:
		// In multi-select mode, <enter>
		// always confirms the current selection.
		if w.multiSelect {
			handled = true
			w.handleSelection()
		}

	case tcell.KeyRune:
		handled = true
		w.mu.Lock()

		r := ek.Rune()
		// Per the documentation of EventKey, it may report the rune
		// 'A' without the ModShift modifier set.
		if unicode.IsUpper(r) {
			r = unicode.ToLower(r)
			w.shiftDown = true
		} else {
			w.shiftDown = ek.Modifiers()&tcell.ModShift != 0
		}

		w.input += string(r)
		defer w.inputChanged()
		w.mu.Unlock()
	}

	return handled
}

func (w *Widget) inputChanged() {
	// We use prefix-free hint labels, so if a label matches the input
	// exactly, we have a guarantee that this is a match.
	defer w.annotateText()

	w.mu.Lock()
	idx, ok := w.hintsByLabel[w.input]
	if ok {
		h := w.hints[idx]
		h.Selected = !h.Selected // toggle selection
		w.hints[idx] = h

		// Clear the input to allow for more selections
		// if we're in multi-select mode.
		w.input = ""
	}
	w.mu.Unlock()

	// If we're not in multi-select mode,
	// we can report the selection immediately.
	if ok && !w.multiSelect {
		w.handleSelection()
	}
}

func (w *Widget) handleSelection() {
	matchers := make(map[string]struct{})
	var (
		text  strings.Builder
		count int
	)
	for idx, h := range w.hints {
		if !h.Selected {
			continue
		}
		if count > 0 {
			text.WriteString(" ")
		}
		count++
		text.WriteString(h.Text)
		for _, m := range h.Matches {
			matchers[m.Matcher] = struct{}{}
		}

		// Deselect the hint in the widget
		// in case we want to select it again.
		//
		// This typically won't happen because HandleSelection upstream
		// will exit the UI loop,
		// but there's no guarantee of that for the Widget interface.
		h.Selected = false
		w.hints[idx] = h
	}

	if count == 0 {
		// There were no matches selected.
		// This is a no-op.
		return
	}

	sel := Selection{
		Text:  text.String(),
		Shift: w.shiftDown,
	}
	for m := range matchers {
		sel.Matchers = append(sel.Matchers, m)
	}
	sort.Strings(sel.Matchers)

	w.handler.HandleSelection(sel)
}

func (w *Widget) annotateText() {
	w.mu.Lock()
	defer w.mu.Unlock()

	var anns []ui.TextAnnotation
	for _, hint := range w.hints {
		input := w.input
		style := AnnotationStyle{
			Match:      w.style.Match,
			Skipped:    w.style.SkippedMatch,
			Label:      w.style.HintLabel,
			LabelTyped: w.style.HintLabelInput,
		}

		if hint.Selected {
			// If this hint is selected, we're in multi-select mode,
			// and we want to allow deselection.
			//
			// Pretend there's no input,
			// and use the DeselectLabel style for hints.
			input = ""
			style.Match = w.style.SelectedMatch
			style.Label = w.style.DeselectLabel
		}

		anns = append(anns, hint.Annotations(input, style)...)
	}

	w.textw.SetAnnotations(anns...)
}
