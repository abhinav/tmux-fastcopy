package main

import (
	"bytes"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestVersion(t *testing.T) {
	t.Parallel()

	var buff bytes.Buffer
	err := (&mainCmd{
		Stdout: &buff,
	}).Run([]string{"-version"})
	td.CmpNoError(t, err)
	td.CmpContains(t, buff.String(), _version)
}
