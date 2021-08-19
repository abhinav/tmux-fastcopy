package main

import (
	"flag"
	"math/rand"
	"testing"
	"testing/quick"
	"time"

	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxopt"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/maxatome/go-testdeep/td"
)

func TestConfigFlags(t *testing.T) {
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
			desc: "action",
			give: []string{"-action", "pbcopy"},
			want: config{Action: "pbcopy"},
		},
		{
			desc: "alphabet",
			give: []string{"-alphabet", "0123456789"},
			want: config{Alphabet: "0123456789"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			var cfg config
			fset := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
			cfg.RegisterFlags(fset)

			td.CmpNoError(t, fset.Parse(tt.give))
			td.Cmp(t, cfg, tt.want)

			t.Run("args", func(t *testing.T) {
				args := cfg.Flags()

				fset := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
				var got config
				got.RegisterFlags(fset)

				if !td.CmpNoError(t, fset.Parse(args)) {
					return
				}

				td.Cmp(t, got, cfg)
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
			give: `@fastcopy-action "tmux set-buffer -- {}"`,
			want: config{Action: "tmux set-buffer -- {}"},
		},
		{
			desc: "alphabet",
			give: "@fastcopy-alphabet abc",
			want: config{Alphabet: "abc"},
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
			td.CmpNoError(t, err)
			td.Cmp(t, got, tt.want)
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
				},
				{
					Pane:     "ignored",
					Action:   "ignored",
					Alphabet: "ignored",
				},
			},
			want: config{
				Pane:     "foo",
				Action:   "bar",
				Alphabet: "abc",
				Verbose:  true,
			},
		},
		{
			desc: "partial merge",
			give: []config{
				{Pane: "foo"},
				{Action: "bar"},
				{Alphabet: "abc"},
				{Verbose: true},
			},
			want: config{
				Pane:     "foo",
				Action:   "bar",
				Alphabet: "abc",
				Verbose:  true,
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

			td.Cmp(t, got, tt.want)
		})
	}
}

func TestConfigFlagsQuickCheck(t *testing.T) {
	// Make sure that config is always round-trippable because we need to
	// the wrapper process to be able to send the exact same configuration
	// down to the wrapped process.

	seed := time.Now().UnixNano()
	defer func() {
		if t.Failed() {
			t.Logf("random seed: %v", seed)
		}
	}()

	random := rand.New(rand.NewSource(seed))
	quick.Check(func(give config) bool {
		// Skip invalid alphabets.
		if give.Alphabet.Validate() != nil {
			return true
		}

		flag := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
		var got config
		got.RegisterFlags(flag)

		if !td.CmpNoError(t, flag.Parse(give.Flags())) {
			return false
		}

		return td.Cmp(t, got, give)
	}, &quick.Config{Rand: random})
}

func TestUsageHasAllConfigFlags(t *testing.T) {
	// We use _usage to write the user facing help. Make sure that every
	// flag registered by RegisterFlags has a corresponding entry in
	// _usage.

	fset := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
	new(config).RegisterFlags(fset)

	fset.VisitAll(func(f *flag.Flag) {
		td.Cmp(t, _usage, td.Contains("\t-"+f.Name),
			"flag %q should be documented", f.Name)
	})

}
