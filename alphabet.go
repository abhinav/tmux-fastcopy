package main

import (
	"errors"
	"fmt"
	"sort"
)

const _defaultAlphabet = "abcdefghijklmnopqrstuvwxyz"

func validateAlphabet(alpha string) error {
	if len(alpha) < 2 {
		return errors.New("alphabet must have at least two items")
	}

	seen := make(map[rune]struct{}, len(alpha))
	dupes := make(map[rune]struct{})
	for _, r := range alpha {
		if _, ok := seen[r]; ok {
			dupes[r] = struct{}{}
		}
		seen[r] = struct{}{}
	}

	if len(dupes) == 0 {
		return nil
	}

	dlist := make([]rune, 0, len(dupes))
	for r := range dupes {
		dlist = append(dlist, r)
	}
	sort.Slice(dlist, func(i, j int) bool {
		return dlist[i] < dlist[j]
	})

	return fmt.Errorf("alphabet has duplicates: %q", dlist)
}
