package main

import (
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestRegexpMatcher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc     string
		regex    string
		s        string
		wantSel  []string
		wantFull []string
	}{
		{
			desc:     "empty",
			s:        "foo",
			wantSel:  []string{},
			wantFull: []string{},
		},
		{
			desc:     "full match",
			regex:    "a(?:b|c)",
			s:        "foo ab bar ac",
			wantSel:  []string{"ab", "ac"},
			wantFull: []string{"ab", "ac"},
		},
		{
			desc:     "subexp match",
			regex:    "a(b|c)",
			s:        "foo ab bar ac",
			wantSel:  []string{"b", "c"},
			wantFull: []string{"ab", "ac"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			m, err := compileRegexpMatcher(tt.desc, tt.regex)
			if !td.CmpNoError(t, err, "compile regex") {
				return
			}

			td.CmpNotPanic(t, func() {
				_ = m.String()
			}, "String")

			t.Run("Name", func(t *testing.T) {
				td.Cmp(t, m.Name(), tt.desc)
			})

			t.Run("Matches", func(t *testing.T) {
				ms := m.AppendMatches(tt.s, nil)
				gotSel := make([]string, len(ms))
				gotFull := make([]string, len(ms))
				for i, m := range ms {
					gotSel[i] = tt.s[m.Sel.Start:m.Sel.End]
					gotFull[i] = tt.s[m.Full.Start:m.Full.End]
				}

				td.Cmp(t, gotSel, tt.wantSel)
				td.Cmp(t, gotFull, tt.wantFull)
			})
		})
	}
}
