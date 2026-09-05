package main

import (
	"errors"
	"flag"
	"fmt"
	"slices"
)

const _defaultAlphabet alphabet = "abcdefghijklmnopqrstuvwxyz"

type alphabet string

var _ flag.Value = (*alphabet)(nil)

func (al *alphabet) String() string {
	return string(*al)
}

func (al *alphabet) Set(alpha string) error {
	*al = alphabet(alpha)
	return al.Validate()
}

func (al alphabet) Validate() error {
	if len(al) < 2 {
		return errors.New("alphabet must have at least two items")
	}

	seen := make(map[rune]struct{}, len(al))
	dupes := make(map[rune]struct{})
	for _, r := range al {
		if _, ok := seen[r]; ok {
			dupes[r] = struct{}{}
		}
		seen[r] = struct{}{}
	}

	if len(dupes) == 0 {
		return nil // success
	}

	dlist := make([]rune, 0, len(dupes))
	for r := range dupes {
		dlist = append(dlist, r)
	}
	slices.Sort(dlist)

	return fmt.Errorf("alphabet has duplicates: %q", dlist)
}
