package main

import (
	"flag"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"
	"unicode/utf8"

	"github.com/abhinav/tmux-fastcopy/internal/iotest"
	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxopt"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatcherDefaultRegexes(t *testing.T) {
	t.Parallel()

	matcher := make(matcher, 0, len(_defaultRegexes))
	for name, reg := range _defaultRegexes {
		m, err := compileRegexpMatcher(name, reg)
		require.NoError(t, err, "compile %q (%q)", name, reg)
		matcher = append(matcher, m)
	}

	tests := []struct {
		desc string
		give string
		want []string
	}{
		{
			desc: "ipv4",
			give: "there's no place like 127.0.0.1",
			want: []string{"127.0.0.1"},
		},
		{
			desc: "gitsha/short",
			give: "commit 016ca97 (origin/main, main)",
			want: []string{"016ca97"},
		},
		{
			desc: "gitsha/long",
			give: "This reverts commit dbf2bb40bf8711e5d854c22d8bf19fc58da38cf2.",
			want: []string{"dbf2bb40bf8711e5d854c22d8bf19fc58da38cf2"},
		},
		{
			desc: "panic", // numbers, addresses, paths
			give: joinLines(
				"goroutine 36 [running]:",
				"testing.tRunner.func1.2(0x1265c60, 0x13052c8)",
				"        /usr/local/Cellar/go/1.16.6/libexec/src/testing/testing.go:1143 +0x332",
			),
			want: []string{
				"0x1265c60", "0x13052c8", "0x332",
				"1143",
				"/usr/local/Cellar/go/1.16.6/libexec/src/testing/testing.go",
			},
		},
		{
			desc: "hexcolor/short",
			give: "background-color: #eee",
			want: []string{"#eee"},
		},
		{
			desc: "hexcolor/long",
			give: "background-color: #f8f8f0;",
			want: []string{"#f8f8f0"},
		},
		{
			desc: "uuid/upper",
			give: "A13BBDE2-2FAB-40A3-B00C-949AC6EBDD79",
			want: []string{"A13BBDE2-2FAB-40A3-B00C-949AC6EBDD79"},
		},
		{
			desc: "uuid/lower",
			give: "425a6a91-58aa-4027-8940-feecaaaece02",
			want: []string{
				"425a6a91-58aa-4027-8940-feecaaaece02",
				// lower case UUID overlaps with other number
				// and gitsha:
				//   "425a6a91" "-4027" "-8940" "feecaaaece02"
			},
		},
		{
			desc: "date",
			give: "2021-08-14 12:34 -0700",
			want: []string{"2021-08-14", "-0700"},
		},
		{
			desc: "path/url overlap",
			give: "http://example.com/foo/bar/baz",
			want: []string{}, // no match
		},
		{
			desc: "path/start of line",
			give: "foo/bar/baz",
			want: []string{"foo/bar/baz"}, // no match
		},
		{
			desc: "path/boundary",
			give: "path=foo/bar/baz",
			want: []string{"foo/bar/baz"}, // no match
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			var got []string
			for _, m := range matcher.Match(tt.give) {
				got = append(got, tt.give[m.Start:m.End])
			}

			assert.ElementsMatch(t, tt.want, got)
		})
	}
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
			fset.SetOutput(iotest.Writer(t))
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
				fset.SetOutput(iotest.Writer(t))
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
					Pane:     "ignored",
					Action:   "ignored",
					Alphabet: "ignored",
					LogFile:  "ignored.txt",
					Tmux:     "/usr/bin/tmux",
					Regexes: regexes{
						"foo": "ignored",
						"bar": "baz",
					},
				},
			},
			want: config{
				Pane:     "foo",
				Action:   "bar",
				Alphabet: "abc",
				Verbose:  true,
				LogFile:  "foo.txt",
				Tmux:     "/usr/bin/tmux",
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
			},
			want: config{
				Pane:     "foo",
				Action:   "bar",
				Alphabet: "abc",
				Verbose:  true,
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

func TestConfigFlagsQuickCheck(t *testing.T) {
	t.Parallel()

	// Make sure that config is always round-trippable because we need to
	// the wrapper process to be able to send the exact same configuration
	// down to the wrapped process.

	seed := time.Now().UnixNano()
	defer func() {
		if t.Failed() {
			t.Logf("random seed: %v", seed)
		}
	}()

	quick.Check(func(give config) bool {
		// Skip invalid alphabets.
		if give.Alphabet.Validate() != nil {
			return true
		}

		flag := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
		flag.SetOutput(iotest.Writer(t))
		var got config
		got.RegisterFlags(flag)

		if !assert.NoError(t, flag.Parse(give.Flags())) {
			return false
		}

		if len(give.Regexes) == 0 {
			give.Regexes = nil // to make nil v non-nil map comparison easier
		}

		return assert.Equal(t, give, got)
	}, &quick.Config{
		Rand: rand.New(rand.NewSource(seed)),
		Values: func(vs []reflect.Value, rand *rand.Rand) {
			vs[0] = reflect.ValueOf(generateConfig(t, rand))
		},
	})
}

func TestUsageHasAllConfigFlags(t *testing.T) {
	t.Parallel()

	// We use _usage to write the user facing help. Make sure that every
	// flag registered by RegisterFlags has a corresponding entry in
	// _usage.

	fset := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
	fset.SetOutput(iotest.Writer(t))
	new(config).RegisterFlags(fset)

	fset.VisitAll(func(f *flag.Flag) {
		assert.Contains(t, _usage, "\t-"+f.Name,
			"flag %q should be documented", f.Name)
	})

}

var (
	_typeString = reflect.TypeOf("")
	_typeBool   = reflect.TypeOf(true)
)

func generateConfig(t testing.TB, rand *rand.Rand) config {
	return config{
		Pane:     generateString(t, rand, 0, "generate pane"),
		Action:   generateString(t, rand, 0, "generate action"),
		Alphabet: generateAlphabet(t, rand),
		Verbose:  generateValue(t, rand, _typeBool, "verbose").(bool),
		Regexes:  generateRegexes(t, rand),
		LogFile:  generateString(t, rand, 0, "generate logFile"),
		Tmux:     generateString(t, rand, 1, "generate tmux"),
	}
}

func generateAlphabet(t testing.TB, rand *rand.Rand) alphabet {
	for {
		alpha := generateString(t, rand, 2, "generate alphabet")

		runes := make(map[rune]struct{})
		for _, r := range alpha {
			runes[r] = struct{}{}
		}

		if len(runes) < 2 {
			continue // too short, try again
		}

		out := make([]rune, 0, len(runes))
		for r := range runes {
			out = append(out, r)
		}
		return alphabet(out)
	}
}

func generateRegexes(t testing.TB, rand *rand.Rand) regexes {
	count := rand.Intn(100)
	m := make(regexes, count)
	for i := 0; i < count; i++ {
		var key string
		for {
			key = generateString(t, rand, 1, "regex name")
			if strings.IndexByte(key, ':') < 0 {
				break
			}
		}
		value := generateString(t, rand, 1, "regex value")
		m[key] = value
	}
	return m
}

func generateString(t testing.TB, rand *rand.Rand, minLength int, msg ...interface{}) string {
	for {
		s := generateValue(t, rand, _typeString, msg...).(string)
		if len(s) < minLength {
			continue
		}
		if !utf8.ValidString(s) {
			continue
		}
		return s
	}
}

func generateValue(t testing.TB, rand *rand.Rand, typ reflect.Type, msg ...interface{}) interface{} {
	v, ok := quick.Value(typ, rand)
	require.True(t, ok, msg...)
	return v.Interface()
}

func joinLines(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}
