package ui

import (
	"fmt"
	"sort"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

// TextAnnotation changes what gets rendered for AnnotatedText.
type TextAnnotation interface {
	offset() int
	length() int
}

// StyleTextAnnotation changes the style of a section of text in AnnotatedText.
type StyleTextAnnotation struct {
	Style tcell.Style // style for this section

	// Offset in the text, and the length of it for which this alternative
	// style applies.
	Offset, Length int
}

func (sa StyleTextAnnotation) offset() int { return sa.Offset }
func (sa StyleTextAnnotation) length() int { return sa.Length }

// OverlayTextAnnotation overlays a different text over a section of text in
// AnnotatedText.
type OverlayTextAnnotation struct {
	Overlay string
	Style   tcell.Style // style for the overlay

	// Offset in the text over which to draw this overlay.
	Offset int
}

func (oa OverlayTextAnnotation) offset() int { return oa.Offset }
func (oa OverlayTextAnnotation) length() int { return len(oa.Overlay) }

// AnnotatedText is a block of text rendered with annotations.
type AnnotatedText struct {
	// Text block to render. This may be multi-line.
	Text  string
	Style tcell.Style

	mu   sync.RWMutex
	anns []TextAnnotation // sorted by offset
}

var _ Widget = (*AnnotatedText)(nil)

// SetAnnotations changes the annotations for an AnnotatedText. Offsets MUST
// not overlap.
func (at *AnnotatedText) SetAnnotations(anns ...TextAnnotation) {
	anns = append(make([]TextAnnotation, 0, len(anns)), anns...)
	sort.Sort(byOffset(anns))

	at.mu.Lock()
	at.anns = anns
	at.mu.Unlock()
}

// Draw draws the annotated text onto the provided view.
func (at *AnnotatedText) Draw(view views.View) {
	at.mu.RLock()
	defer at.mu.RUnlock()

	var (
		lastIdx int
		pos     Pos
	)
	for _, ann := range at.anns {
		if ann.length() == 0 {
			continue
		}

		// Previous annotation overlaps with this one. Skip.
		if ann.offset() < lastIdx {
			continue
		}

		// TODO: The way this is set up, an overlay annotation
		// can undo the row increment that would happen from a newline.
		// This is probably not the best internal representation.

		pos = DrawText(at.Text[lastIdx:ann.offset()], at.Style, view, pos)

		var (
			style tcell.Style
			text  string
		)
		switch ann := ann.(type) {
		case StyleTextAnnotation:
			style = ann.Style
			text = at.Text[ann.Offset : ann.Offset+ann.Length]

		case OverlayTextAnnotation:
			style = ann.Style
			text = ann.Overlay

		default:
			panic(fmt.Sprintf("unknown annotation %#v", ann))
		}

		pos = DrawText(text, style, view, pos)
		lastIdx = ann.offset() + ann.length()
	}

	DrawText(at.Text[lastIdx:], at.Style, view, pos)
}

// HandleEvent returns false.
func (at *AnnotatedText) HandleEvent(tcell.Event) bool {
	return false
}

// byOffset sorts TextAnnotations by offset.
type byOffset []TextAnnotation

func (as byOffset) Len() int { return len(as) }

func (as byOffset) Swap(i, j int) {
	as[i], as[j] = as[j], as[i]
}

func (as byOffset) Less(i, j int) bool {
	return as[i].offset() < as[j].offset()
}
