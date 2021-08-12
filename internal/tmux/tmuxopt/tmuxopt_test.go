package tmuxopt

import (
	"strings"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/maxatome/go-testdeep/td"
)

func TestLoaderStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc    string
		give    []byte   // tmux response
		options []string // string options to request
		want    []string // values for those options in-order
	}{
		{
			desc: "empty",
			want: []string{},
		},
		{
			desc: "simple values",
			give: unlines(
				"foo bar",
				"baz qux",
				"qux quux",
			),
			options: []string{"foo", "qux"},
			want:    []string{"bar", "quux"},
		},
		{
			desc: "skip bad lines",
			give: unlines(
				"a b",
				"",
				"cde",
				"f g",
			),
			options: []string{"a", "c", "f"},
			want:    []string{"b", "", "g"},
		},
		{
			desc: "unquote/single quote",
			give: unlines(
				"prefix ' '",
			),
			options: []string{"prefix"},
			want:    []string{" "},
		},
		{
			desc: "unquote/double quote",
			give: unlines(
				`command "tmux set-buffer -- {}"`,
			),
			options: []string{"command"},
			want:    []string{"tmux set-buffer -- {}"},
		},
		{
			desc: "unquote/escape",
			give: unlines(
				`foo "bar \" baz"`,
			),
			options: []string{"foo"},
			want:    []string{`bar " baz`},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			if !td.CmpLen(t, tt.want, len(tt.options), "invalid test") {
				return
			}

			ctrl := gomock.NewController(t)
			mockTmux := tmuxtest.NewMockDriver(ctrl)

			loader := Loader{Tmux: mockTmux}
			got := make([]string, len(tt.options))
			for i, opt := range tt.options {
				loader.StringVar(&got[i], opt)
			}

			mockTmux.EXPECT().
				ShowOptions(gomock.Any()).
				Return(tt.give, nil).
				AnyTimes()

			err := loader.Load(tmux.ShowOptionsRequest{})
			td.CmpNoError(t, err)

			td.Cmp(t, got, tt.want)
		})
	}
}

func unlines(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n") + "\n")
}
