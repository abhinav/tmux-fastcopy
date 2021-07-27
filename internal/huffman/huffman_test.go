package huffman

import (
	"fmt"
	"math/rand"
	"testing"
	"testing/quick"
	"time"

	"github.com/maxatome/go-testdeep/td"
)

// Index based test cases are difficult to read. Set up some machinery to write
// more readable test cases.
type alphabet struct {
	items   []rune
	reverse map[rune]int // index into items
}

func newAlphabet(chars string) *alphabet {
	items := []rune(chars)
	reverse := make(map[rune]int, len(items))
	for i, r := range items {
		reverse[r] = i
	}
	return &alphabet{items: items, reverse: reverse}
}

func (a *alphabet) String() string {
	return string(a.items)
}

func (a *alphabet) Size() int { return len(a.items) }

func (a *alphabet) Label(indexes []int) string {
	label := make([]rune, len(indexes))
	for i, idx := range indexes {
		label[i] = a.items[idx]
	}
	return string(label)
}

func (a *alphabet) Labels(indexes [][]int) []string {
	labels := make([]string, len(indexes))
	for i, idxes := range indexes {
		labels[i] = a.Label(idxes)
	}
	return labels
}

var (
	// Alphabets used for tests.
	_ab   = newAlphabet("ab")
	_abcd = newAlphabet("abcd")

	_alphabets = []*alphabet{
		_ab,
		_abcd,
	}
)

func TestLabel(t *testing.T) {
	type item struct {
		Freq  int
		Label string
	}

	tests := []struct {
		alphabet *alphabet
		items    []item
	}{
		{
			alphabet: _ab,
			items: []item{
				{1, "aa"},
				{1, "bbb"},
				{1, "ba"},
				{1, "bba"},
				{1, "ab"},
			},
		},
		{
			alphabet: _abcd,
			items: []item{
				{5, "a"},
				{4, "bd"},
				{3, "bc"},
				{2, "bb"},
				{1, "ba"},
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%v/%v", tt.alphabet, i), func(t *testing.T) {
			freqs := make([]int, len(tt.items))
			want := make([]string, len(tt.items))
			for i, item := range tt.items {
				freqs[i] = item.Freq
				want[i] = item.Label
			}

			gotIndexes := Label(tt.alphabet.Size(), freqs)
			got := tt.alphabet.Labels(gotIndexes)
			td.Cmp(t, got, want,
				"Labels(%d, %v)", tt.alphabet.Size(), freqs)

			assertLabelInvariants(t, len(tt.items), got)
		})
	}
}

func TestQuick(t *testing.T) {
	runTest := func(t *testing.T, alphabet *alphabet) interface{} {
		return func(freqs []int) bool {
			got := alphabet.Labels(
				Label(alphabet.Size(), freqs),
			)
			return assertLabelInvariants(t, len(freqs), got)
		}
	}

	for _, alphabet := range _alphabets {
		t.Run(alphabet.String(), func(t *testing.T) {
			seed := time.Now().UnixNano()
			t.Logf("random seed: %v", seed)

			err := quick.Check(
				runTest(t, alphabet),
				&quick.Config{
					Rand: rand.New(rand.NewSource(seed)),
				},
			)
			td.CmpNoError(t, err)
		})
	}
}

func assertLabelInvariants(t *testing.T, numItems int, labels []string) bool {
	t.Helper()

	// 1) Number of labels must match the number of
	//    frequencies/elements.
	if !td.Cmp(t, labels, td.Len(numItems)) {
		return false
	}

	// 2) There must be no duplicates.
	seen := make(map[string]struct{})
	for _, label := range labels {
		if !td.Cmp(t, seen, td.Not(td.ContainsKey(label))) {
			return false
		}
		seen[label] = struct{}{}
	}

	// 3) None of the labels is a prefix for another.
	for i, left := range labels {
		for j, right := range labels {
			if i == j {
				continue
			}
			if !td.Cmp(t, left, td.Not(td.HasPrefix(right))) {
				return false
			}
		}
	}

	return true
}
