package fastcopy

import (
	"sort"
	"strings"

	"github.com/abhinav/tmux-fastcopy/internal/huffman"
	"github.com/abhinav/tmux-fastcopy/internal/ui"
)

type hint struct {
	Label   string
	Text    string
	Matches []Range
}

// generateHints generates a list of hints for the given text. It uses alphabet
// to generate unique prefix-free labels for matche sin the text, where matches
// are defined by the provided ranges.
func generateHints(alphabet []rune, text string, matches []Range) []hint {
	labelFrom := func(indexes []int) string {
		label := make([]rune, len(indexes))
		for i, idx := range indexes {
			label[i] = alphabet[idx]
		}
		return string(label)
	}

	// Grouping of match ranges by their matched text.
	byText := make(map[string][]Range)
	for _, r := range matches {
		match := text[r.Start:r.End]
		byText[match] = append(byText[match], r)
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
func (h *hint) Annotations(input string, style Style) (anns []ui.TextAnnotation) {
	matched := strings.HasPrefix(h.Label, input)

	// If the hint matches the input, overlay the hint (both, typed
	// and non-typed portions) over the string. Otherwise, grey out
	// the match.
	matchStyle := style.SkippedMatch
	if matched {
		matchStyle = style.Match
	}

	for _, pos := range h.Matches {
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
					Style:   style.HintLabelInput,
				})
				i += len(input)
			}

			// Highlight the portion of the label yet to be typed.
			if i < len(h.Label) {
				anns = append(anns, ui.OverlayTextAnnotation{
					Offset:  pos.Start + len(input),
					Overlay: h.Label[i:],
					Style:   style.HintLabel,
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
