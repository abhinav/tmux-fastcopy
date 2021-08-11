package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestConfigParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give []string
		want config
	}{
		{desc: "no args"}, // zero values
		{
			desc: "pane",
			give: []string{"--pane", "42"},
			want: config{Pane: "42"},
		},
		{
			desc: "verbose",
			give: []string{"--verbose"},
			want: config{Verbose: true},
		},
		{
			desc: "log",
			give: []string{"--log", "log.txt"},
			want: config{LogFile: "log.txt"},
		},
		{
			desc: "action",
			give: []string{"-action", "pbcopy"},
			want: config{Action: "pbcopy"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			fset := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
			cfg := newConfig(fset)

			td.CmpNoError(t, fset.Parse(tt.give))
			td.Cmp(t, cfg, &tt.want)

			t.Run("args", func(t *testing.T) {
				args := cfg.Args()

				fset := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
				got := newConfig(fset)

				if !td.CmpNoError(t, fset.Parse(args)) {
					return
				}

				td.Cmp(t, got, cfg)
			})
		})
	}
}

func TestConfigBuildLogWriter(t *testing.T) {
	t.Parallel()

	t.Run("stderr", func(t *testing.T) {
		t.Parallel()

		var (
			cfg  config
			buff bytes.Buffer
		)
		w, closew, err := cfg.BuildLogWriter(&buff)
		if !td.CmpNoError(t, err) {
			return
		}
		defer closew()

		if _, err := io.WriteString(w, "foo"); !td.CmpNoError(t, err) {
			return
		}

		td.Cmp(t, buff.String(), "foo")
	})

	t.Run("file", func(t *testing.T) {
		t.Parallel()

		logFile := filepath.Join(t.TempDir(), "log.out")
		cfg := config{LogFile: logFile}

		var buff bytes.Buffer
		defer func() { td.CmpEmpty(t, buff.String()) }()

		w, closew, err := cfg.BuildLogWriter(&buff)
		if !td.CmpNoError(t, err) {
			return
		}

		if _, err := io.WriteString(w, "foo"); !td.CmpNoError(t, err) {
			return
		}
		closew()

		got, err := ioutil.ReadFile(logFile)
		if td.CmpNoError(t, err) {
			td.Cmp(t, string(got), "foo")
		}
	})

	t.Run("file/open error", func(t *testing.T) {
		t.Parallel()

		logFile := filepath.Join(t.TempDir(), "does/not/exist/log.out")

		cfg := config{LogFile: logFile}
		_, _, err := cfg.BuildLogWriter(io.Discard)
		td.CmpError(t, err)

		_, err = os.Stat(logFile)
		td.CmpError(t, err)
	})
}

func TestUsageHasAllConfigFlags(t *testing.T) {
	// We use _usage to write the user facing help. Make sure that every
	// flag registered by newConfig has a corresponding entry in _usage.

	fset := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
	newConfig(fset)

	fset.VisitAll(func(f *flag.Flag) {
		td.Cmp(t, _usage, td.Contains("\t-"+f.Name),
			"flag %q should be documented", f.Name)
	})

}
