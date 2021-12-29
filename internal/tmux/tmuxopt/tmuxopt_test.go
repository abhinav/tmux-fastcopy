package tmuxopt

import (
	"errors"
	"strings"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/tmux"
	"github.com/abhinav/tmux-fastcopy/internal/tmux/tmuxtest"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			desc: "empty value",
			give: unlines(
				"foo ",
			),
			options: []string{"foo"},
			want:    []string{""},
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
			desc: "unquote/single quote/multiple characters",
			give: unlines(
				"prefix 'a b c'",
			),
			options: []string{"prefix"},
			want:    []string{"a b c"},
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
		{
			desc: "unquote/escape/single-quoted",
			give: unlines(
				`foo 'foo \\" bar'`,
				// == set-option -g foo 'foo \" bar'
			),
			options: []string{"foo"},
			want:    []string{`foo \" bar`},
		},
		{
			// For
			// https://github.com/abhinav/tmux-fastcopy/issues/48.
			// Adding either of the following to your tmux.conf
			// results in '"hello"' in the output.
			//   set-option -g @fastcopy-regex-test1 "\"hello\""
			//   set-option -g @fastcopy-regex-test1 '"hello"'
			desc: "issue48",
			give: unlines(
				`@fastcopy-regex-test1 '"hello"'`,
			),
			options: []string{"@fastcopy-regex-test1"},
			want:    []string{`"hello"`},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			require.Len(t, tt.want, len(tt.options), "invalid test")

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
			require.NoError(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoaderMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc    string
		give    []byte   // tmux response
		options []string // prefixes to request
		want    []map[string]string
	}{
		{
			desc: "simple",
			give: unlines(
				"foo-bar baz",
				"foo-baz qux",
			),
			options: []string{"foo-"},
			want: []map[string]string{
				{
					"bar": "baz",
					"baz": "qux",
				},
			},
		},
		{
			desc: "quoted",
			give: unlines(
				`foo-bar "baz\tqux"`,
			),
			options: []string{"foo-"},
			want: []map[string]string{
				{"bar": "baz\tqux"},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			require.Len(t, tt.want, len(tt.options), "invalid test")

			ctrl := gomock.NewController(t)
			mockTmux := tmuxtest.NewMockDriver(ctrl)

			loader := Loader{Tmux: mockTmux}
			got := make([]map[string]string, len(tt.options))
			for i, opt := range tt.options {
				m := make(map[string]string)
				loader.MapVar(mapVar(m), opt)
				got[i] = m
			}

			mockTmux.EXPECT().
				ShowOptions(gomock.Any()).
				Return(tt.give, nil).
				AnyTimes()

			err := loader.Load(tmux.ShowOptionsRequest{})
			require.NoError(t, err)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoaderShowOptionsError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockTmux := tmuxtest.NewMockDriver(ctrl)

	loader := Loader{Tmux: mockTmux}
	loader.StringVar(new(string), "foo")
	mockTmux.EXPECT().
		ShowOptions(gomock.Any()).
		Return(nil, errors.New("great sadness"))

	err := loader.Load(tmux.ShowOptionsRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "great sadness")
}

func TestLoaderSetError(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockTmux := tmuxtest.NewMockDriver(ctrl)

	loader := Loader{Tmux: mockTmux}
	loader.Var(errorVar{errors.New("great sadness")}, "foo")

	mockTmux.EXPECT().
		ShowOptions(gomock.Any()).
		Return([]byte("foo bar\n"), nil)

	err := loader.Load(tmux.ShowOptionsRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `load option "foo": great sadness`)
}

func unlines(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n") + "\n")
}

type mapVar map[string]string

func (m mapVar) Put(k, v string) error {
	m[k] = v
	return nil
}

type errorVar struct{ err error }

func (e errorVar) Set(string) error { return e.err }
