package log

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give []string
		want string
	}{
		{desc: "empty"},
		{
			desc: "split message",
			give: []string{"foo\nbar"},
			want: unlines(
				"[x] foo",
				"[x] bar",
			),
		},
		{
			desc: "ends with a newline",
			give: []string{"foo\n", "bar\n"},
			want: unlines(
				"[x] foo",
				"[x] bar",
			),
		},
		{
			desc: "no newlines",
			give: []string{"foo", "bar"},
			want: unlines(
				"[x] foobar",
			),
		},
		{
			desc: "newline late",
			give: []string{"foo", "b\nar"},
			want: unlines(
				"[x] foob",
				"[x] ar",
			),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			log := New(&buff).WithName("x")

			w := Writer{Log: log}
			for _, s := range tt.give {
				_, err := io.WriteString(&w, s)
				require.NoError(t, err)
			}
			require.NoError(t, w.Close())

			assert.Equal(t, tt.want, buff.String())
		})
	}
}
