package fastcopy

import (
	"sort"
	"strings"

	"github.com/abhinav/tmux-fastcopy/internal/ui"
	tcell "github.com/gdamore/tcell/v2"
	"go.abhg.dev/algorithm/huffman"
)

type hint struct {
	// Label to select this hint.
	Label string

	// Text that will be copied if this hint is selected.
	Text string

	// List ot matches identified by this hint.
	//
	// Note that a hint may have multiple matches
	// if the same text appears on the screen multiple times,
	// or if the same text matches multiple regexes.
	Matches []Match

	// Selected reports whether this hint is selected.
	//
	// This is only used in multi-selection mode.
	Selected bool
}

// generateHints generates a list of hints for the given text. It uses alphabet
// to generate unique prefix-free labels for matche sin the text, where matches
// are defined by the provided ranges.
func generateHints(alphabet []rune, text string, matches []Match) []hint {
	labelFrom := func(indexes []int) string {
		label := make([]rune, len(indexes))
		for i, idx := range indexes {
			label[i] = alphabet[idx]
		}
		return string(label)
	}

	// Grouping of match ranges by their matched text.
	byText := make(map[string][]Match)
	for _, m := range matches {
		r := m.Range
		match := text[r.Start:r.End]
		byText[match] = append(byText[match], m)
	}

	uniqueMatches := make([]string, 0, len(byText))
	for t := range byText {
		uniqueMatches = append(uniqueMatches, t)
	}
	sort.Strings(uniqueMatches)

	freqs := make([]int, len(uniqueMatches))
	for i, t := range uniqueMatches {
		freqs[i] = len(byText[t])
	}

	hints := make([]hint, len(uniqueMatches))
	for i, labelIxes := range huffman.Label(len(alphabet), freqs) {
		t := uniqueMatches[i]
		hints[i] = hint{
			Label:   labelFrom(labelIxes),
			Text:    t,
			Matches: byText[t],
		}
	}

	return hints
}

// AnnotationStyle is the style of annotations for hints and matched text.
type AnnotationStyle struct {
	// Matched text that is still a candidate for selection.
	Match tcell.Style

	// Matched text that is no longer a candidate for selection.
	Skipped tcell.Style

	// Label that the user must type to select the hint.
	Label tcell.Style

	// Part of a multi-character label that the user has already typed.
	LabelTyped tcell.Style
}

func (h *hint) Annotations(input string, style AnnotationStyle) (anns []ui.TextAnnotation) {
	matched := strings.HasPrefix(h.Label, input)

	// If the hint matches the input, overlay the hint (both, typed
	// and non-typed portions) over the string. Otherwise, grey out
	// the match.
	matchStyle := style.Skipped
	if matched {
		matchStyle = style.Match
	}

	for _, match := range h.Matches {
		pos := match.Range
		// Show the label only if there's no input, or if the input
		// matches all or part of the label.
		if matched {
			i := 0

			// Highlight the portion of the label already typed by
			// the user.
			if len(input) > 0 {
				anns = append(anns, ui.OverlayTextAnnotation{
					Offset:  pos.Start,
					Overlay: input,
					Style:   style.LabelTyped,
				})
				i += len(input)
			}

			// Highlight the portion of the label yet to be typed.
			if i < len(h.Label) {
				anns = append(anns, ui.OverlayTextAnnotation{
					Offset:  pos.Start + len(input),
					Overlay: h.Label[i:],
					Style:   style.Label,
				})
			}

			pos.Start += len(h.Label)
		}

		// Don't show the rest of the matched text if the label is
		// longer than the text.
		if pos.End > pos.Start {
			anns = append(anns, ui.StyleTextAnnotation{
				Offset: pos.Start,
				Length: pos.End - pos.Start,
				Style:  matchStyle,
			})
		}
	}

	return anns
}
