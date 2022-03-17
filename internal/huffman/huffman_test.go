package huffman

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
)

// Index based test cases are difficult to read. Set up some machinery to write
// more readable test cases.
type alphabet struct {
	items []rune
}

func newAlphabet(chars string) *alphabet {
	items := []rune(chars)
	return &alphabet{items: items}
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

func TestLabelAlphabetTooSmall(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		Label(1, []int{1, 2, 3})
	})
}

func TestLabel(t *testing.T) {
	t.Parallel()

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
		},
		{
			alphabet: _ab,
			items: []item{
				{1, "a"},
			},
		},
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
			assert.Equal(t, want, got,
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
			assert.NoError(t, err)
		})
	}
}

func assertLabelInvariants(t *testing.T, numItems int, labels []string) bool {
	t.Helper()

	// 1) Number of labels must match the number of
	//    frequencies/elements.
	if !assert.Len(t, labels, numItems) {
		return false
	}

	// 2) There must be no duplicates.
	var seen []string
	for _, label := range labels {
		if !assert.NotEmpty(t, label, "label with %d items must not be empty", numItems) {
			return false
		}

		if !assert.NotContains(t, seen, label, "duplicate label %q", label) {
			return false
		}
		seen = append(seen, label)
	}

	// 3) None of the labels is a prefix for another.
	for i, left := range labels {
		for j, right := range labels {
			if i == j {
				continue
			}

			prefix := strings.HasPrefix(left, right)
			if !assert.False(t, prefix, "%q is a prefix of %q", right, left) {
				return false
			}
		}
	}

	return true
}
