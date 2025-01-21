package main

import (
	"flag"
	"strings"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxopt"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.abhg.dev/io/ioutil"
	"pgregory.net/rapid"
)

func TestMatcherDefaultRegexes(t *testing.T) {
	t.Parallel()

	matcher := make(matcher, 0, len(_defaultRegexes))
	for name, reg := range _defaultRegexes {
		m, err := compileRegexpMatcher(name, reg)
		require.NoError(t, err, "compile %q (%q)", name, reg)
		matcher = append(matcher, m)
	}

	type match struct{ Matcher, Value string }

	tests := []struct {
		desc string
		give string
		want []match
	}{
		{
			desc: "ipv4",
			give: "there's no place like 127.0.0.1",
			want: []match{
				{"ipv4", "127.0.0.1"},
			},
		},
		{
			desc: "gitsha/short",
			give: "commit 016ca97 (origin/main, main)",
			want: []match{
				{"gitsha", "016ca97"},
			},
		},
		{
			desc: "gitsha/long",
			give: "This reverts commit dbf2bb40bf8711e5d854c22d8bf19fc58da38cf2.",
			want: []match{
				{"gitsha", "dbf2bb40bf8711e5d854c22d8bf19fc58da38cf2"},
			},
		},
		{
			desc: "panic", // numbers, addresses, paths
			give: joinLines(
				"goroutine 36 [running]:",
				"testing.tRunner.func1.2(0x1265c60, 0x13052c8)",
				"        /usr/local/Cellar/go/1.16.6/libexec/src/testing/testing.go:1143 +0x332",
			),
			want: []match{
				{"hexaddr", "0x1265c60"},
				{"hexaddr", "0x13052c8"},
				{"hexaddr", "0x332"},
				{"int", "1143"},
				{"path", "/usr/local/Cellar/go/1.16.6/libexec/src/testing/testing.go"},
			},
		},
		{
			desc: "hexcolor/short",
			give: "background-color: #eee",
			want: []match{
				{"hexcolor", "#eee"},
			},
		},
		{
			desc: "hexcolor/long",
			give: "background-color: #f8f8f0;",
			want: []match{
				{"hexcolor", "#f8f8f0"},
			},
		},
		{
			desc: "uuid/upper",
			give: "A13BBDE2-2FAB-40A3-B00C-949AC6EBDD79",
			want: []match{
				{"uuid", "A13BBDE2-2FAB-40A3-B00C-949AC6EBDD79"},
			},
		},
		{
			desc: "uuid/lower",
			give: "425a6a91-58aa-4027-8940-feecaaaece02",
			want: []match{
				{"uuid", "425a6a91-58aa-4027-8940-feecaaaece02"},
				// lower case UUID overlaps with other number
				// and gitsha:
				//   "425a6a91" "-4027" "-8940" "feecaaaece02"
			},
		},
		{
			desc: "date",
			give: "2021-08-14 12:34 -0700",
			want: []match{
				{"isodate", "2021-08-14"},
				{"int", "-0700"},
			},
		},
		{
			desc: "path/url overlap",
			give: "http://example.com/foo/bar/baz",
			want: []match{}, // no match
		},
		{
			desc: "path/start of line",
			give: "foo/bar/baz",
			want: []match{
				{"path", "foo/bar/baz"},
			},
		},
		{
			desc: "path/boundary",
			give: "path=foo/bar/baz",
			want: []match{
				{"path", "foo/bar/baz"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			var got []match
			for _, m := range matcher.Match(tt.give) {
				r := m.Range
				got = append(got, match{m.Matcher, tt.give[r.Start:r.End]})
			}

			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	t.Parallel()

	cfg := config{
		Tmux: "tmux",
		Regexes: regexes{
			"prompt": "^% (.+)$",
		},
	}
	cfg.FillFrom(defaultConfig(&cfg))

	assert.Equal(t, "tmux load-buffer -", cfg.Action)
	assert.Empty(t, cfg.ShiftAction)
	assert.Equal(t, _defaultAlphabet, cfg.Alphabet)

	for k, v := range _defaultRegexes {
		assert.Equal(t, v, cfg.Regexes[k], "regex %q", k)
	}
	assert.Equal(t, "^% (.+)$", cfg.Regexes["prompt"])
}

func TestConfigFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc    string
		give    []string
		want    config
		wantErr string
	}{
		{
			desc: "no args",
			want: config{Tmux: "tmux"},
		},
		{
			desc: "pane",
			give: []string{"--pane", "42"},
			want: config{Pane: "42", Tmux: "tmux"},
		},
		{
			desc: "verbose",
			give: []string{"--verbose"},
			want: config{Verbose: true, Tmux: "tmux"},
		},
		{
			desc: "action",
			give: []string{"-action", "pbcopy"},
			want: config{Action: "pbcopy", Tmux: "tmux"},
		},
		{
			desc: "shift action",
			give: []string{"-shift-action", "open"},
			want: config{ShiftAction: "open", Tmux: "tmux"},
		},
		{
			desc: "alphabet",
			give: []string{"-alphabet", "0123456789"},
			want: config{Alphabet: "0123456789", Tmux: "tmux"},
		},
		{
			desc:    "alphabet/too small",
			give:    []string{"-alphabet", "a"},
			wantErr: `alphabet must have at least two items`,
		},
		{
			desc:    "alphabet/duplicate",
			give:    []string{"-alphabet", "aaa"},
			wantErr: "alphabet has duplicates",
		},
		{
			desc: "regex/single",
			give: []string{"-regex", "foo:bar"},
			want: config{
				Regexes: regexes{"foo": "bar"},
				Tmux:    "tmux",
			},
		},
		{
			desc: "regex/multiple",
			give: []string{
				"-regex", "foo:bar",
				"-regex", "baz:qux",
			},
			want: config{
				Regexes: regexes{
					"foo": "bar",
					"baz": "qux",
				},
				Tmux: "tmux",
			},
		},
		{
			desc:    "regex/no name",
			give:    []string{"-regex", ":bar"},
			wantErr: `regex must have a name`,
		},
		{
			desc: "regex/no regex",
			give: []string{"-regex", "bar:"},
			want: config{
				Regexes: regexes{"bar": ""},
				Tmux:    "tmux",
			},
		},
		{
			desc:    "regex/wrong form",
			give:    []string{"-regex", "foo"},
			wantErr: `must be in the form NAME:REGEX`,
		},
		{
			desc: "log",
			give: []string{"-log", "foo.txt"},
			want: config{LogFile: "foo.txt", Tmux: "tmux"},
		},
		{
			desc: "tmux",
			give: []string{"-tmux", "/usr/bin/tmux"},
			want: config{Tmux: "/usr/bin/tmux"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			var cfg config
			fset := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
			fset.SetOutput(ioutil.TestLogWriter(t, ""))
			cfg.RegisterFlags(fset)

			err := fset.Parse(tt.give)
			if len(tt.wantErr) > 0 {
				require.Error(t, err, "parse failure")
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, cfg, "parse flags")

			// The following turns the parsec cfg back into flags
			// and tries again. This is worth trying only if an
			// error was not expected in the original parse.
			t.Run("Flags", func(t *testing.T) {
				args := cfg.Flags()

				fset := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
				fset.SetOutput(ioutil.TestLogWriter(t, ""))
				var got config
				got.RegisterFlags(fset)

				require.NoError(t, fset.Parse(args))

				assert.Equal(t, cfg, got)
			})
		})
	}
}

func TestConfigTmuxOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give string // output of tmux show-options
		want config
	}{
		{
			desc: "action",
			give: "@fastcopy-action pbcopy",
			want: config{Action: "pbcopy"},
		},
		{
			desc: "action quoted",
			give: `@fastcopy-action "tmux load-buffer -"`,
			want: config{Action: "tmux load-buffer -"},
		},
		{
			desc: "shift-action",
			give: "@fastcopy-shift-action open",
			want: config{ShiftAction: "open"},
		},
		{
			desc: "alphabet",
			give: "@fastcopy-alphabet abc",
			want: config{Alphabet: "abc"},
		},
		{
			desc: "regexes",
			give: joinLines(
				// tmux show-options will escape the '\' in the
				// string.
				`@fastcopy-regex-phab-diff "D\\d{3,}"`,
				`@fastcopy-regex-github-pr "github.com/\\w+/\\w+/pull/\\d+"`,
				`@fastcopy-regex-hexcolor ""`,
			),
			want: config{
				Regexes: regexes{
					"phab-diff": `D\d{3,}`,
					"github-pr": `github.com/\w+/\w+/pull/\d+`,
					"hexcolor":  "",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockTmux := tmuxtest.NewMockDriver(ctrl)

			loader := tmuxopt.Loader{Tmux: mockTmux}

			var got config
			got.RegisterOptions(&loader)

			mockTmux.EXPECT().
				ShowOptions(gomock.Any()).
				Return([]byte(tt.give), nil)

			err := loader.Load(tmux.ShowOptionsRequest{})
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigMerge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give []config
		want config
	}{
		{
			desc: "fill all",
			give: []config{
				{
					Pane:     "foo",
					Action:   "bar",
					Alphabet: "abc",
					Verbose:  true,
					LogFile:  "foo.txt",
					Regexes: regexes{
						"foo": "bar",
					},
				},
				{
					Pane:        "ignored",
					Action:      "ignored",
					ShiftAction: "open",
					Alphabet:    "ignored",
					LogFile:     "ignored.txt",
					Tmux:        "/usr/bin/tmux",
					Regexes: regexes{
						"foo": "ignored",
						"bar": "baz",
					},
				},
			},
			want: config{
				Pane:        "foo",
				Action:      "bar",
				ShiftAction: "open",
				Alphabet:    "abc",
				Verbose:     true,
				LogFile:     "foo.txt",
				Tmux:        "/usr/bin/tmux",
				Regexes: regexes{
					"foo": "bar",
					"bar": "baz",
				},
			},
		},
		{
			desc: "partial merge",
			give: []config{
				{Pane: "foo"},
				{Action: "bar"},
				{Alphabet: "abc"},
				{Verbose: true},
				{Regexes: regexes{"foo": "bar"}},
				{Regexes: regexes{"bar": "baz"}},
				{Regexes: regexes{"foo": "ignored"}},
				{Regexes: regexes{"bar": "ignored"}},
				{LogFile: "foo.txt"},
				{Tmux: "/usr/local/bin/tmux"},
				{ShiftAction: "open"},
			},
			want: config{
				Pane:        "foo",
				Action:      "bar",
				ShiftAction: "open",
				Alphabet:    "abc",
				Verbose:     true,
				Regexes: regexes{
					"foo": "bar",
					"bar": "baz",
				},
				LogFile: "foo.txt",
				Tmux:    "/usr/local/bin/tmux",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			var got config
			for _, c := range tt.give {
				got.FillFrom(&c)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigFlags_rapid(t *testing.T) {
	t.Parallel()

	// Make sure that config is always round-trippable because we need to
	// the wrapper process to be able to send the exact same configuration
	// down to the wrapped process.

	gen := configGenerator()
	rapid.Check(t, func(t *rapid.T) {
		give := gen.Draw(t, "config")

		// Skip invalid alphabets.
		if give.Alphabet.Validate() != nil {
			t.Skip()
		}

		output, done := ioutil.PrintfWriter(t.Logf, "")
		defer done()
		flag := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
		flag.SetOutput(output)

		var got config
		got.RegisterFlags(flag)

		require.NoError(t, flag.Parse(give.Flags()))

		if len(give.Regexes) == 0 {
			give.Regexes = nil // to make nil v non-nil map comparison easier
		}
		require.Equal(t, give, got)
	})
}

func TestUsageHasAllConfigFlags(t *testing.T) {
	t.Parallel()

	// We use _usage to write the user facing help. Make sure that every
	// flag registered by RegisterFlags has a corresponding entry in
	// _usage.

	fset := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
	fset.SetOutput(ioutil.TestLogWriter(t, ""))
	new(config).RegisterFlags(fset)

	fset.VisitAll(func(f *flag.Flag) {
		assert.Contains(t, _usage, "\t-"+f.Name,
			"flag %q should be documented", f.Name)
	})
}

func configGenerator() *rapid.Generator[config] {
	alphabetGen := rapid.Custom(func(t *rapid.T) alphabet {
		alpha := rapid.SliceOfNDistinct(rapid.Rune(), 2, -1, rapid.ID[rune]).Draw(t, "alphabet")
		return alphabet(alpha)
	})

	regexGen := rapid.MapOf(
		rapid.StringN(1, -1, -1).Filter(func(s string) bool {
			return !strings.Contains(s, ":")
		}),
		rapid.StringN(1, -1, -1),
	)

	return rapid.Custom(func(t *rapid.T) config {
		return config{
			Pane:        rapid.String().Draw(t, "pane"),
			Action:      rapid.String().Draw(t, "action"),
			ShiftAction: rapid.String().Draw(t, "shift action"),
			Alphabet:    alphabetGen.Draw(t, "alphabet"),
			Verbose:     rapid.Bool().Draw(t, "verbose"),
			Regexes:     regexGen.Draw(t, "regexes"),
			LogFile:     rapid.String().Draw(t, "logFile"),
			Tmux:        rapid.StringN(1, -1, -1).Draw(t, "tmux"),
		}
	})
}

func joinLines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}
