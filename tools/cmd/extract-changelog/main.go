package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	cmd := mainCmd{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	if err := cmd.Run(os.Args[1:]); err != nil && err != flag.ErrHelp {
		fmt.Fprintln(cmd.Stderr, err)
		os.Exit(1)
	}
}

type mainCmd struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

const _usage = `USAGE: %v [OPTIONS] VERSION
`

func (cmd *mainCmd) Run(args []string) error {
	flag := flag.NewFlagSet("extract-changelog", flag.ContinueOnError)
	flag.SetOutput(cmd.Stderr)
	flag.Usage = func() {
		fmt.Fprintf(flag.Output(), _usage, flag.Name())
		flag.PrintDefaults()
	}

	if err := flag.Parse(args); err != nil {
		return err
	}
	args = flag.Args()

	if len(args) == 0 {
		return errors.New("please provide a version")
	}

	version := strings.TrimPrefix(args[0], "v")

	f, err := os.Open("CHANGELOG.md")
	if err != nil {
		return err
	}
	defer f.Close()

	return extract(f, cmd.Stdout, version)
}

func extract(r io.Reader, w io.Writer, version string) error {
	type _state int

	const (
		initial _state = iota
		printing
	)

	var state _state

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		switch state {
		case initial:
			if strings.HasPrefix(line, "## "+version+" ") {
				fmt.Fprintln(w, line)
				state = printing
			}

		case printing:
			if strings.HasPrefix(line, "## ") {
				return nil
			}
			fmt.Fprintln(w, line)

		default:
			return fmt.Errorf("unexpected state at %q", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	if state < printing {
		return fmt.Errorf("changelog for %q not found", version)
	}
	return nil
}
