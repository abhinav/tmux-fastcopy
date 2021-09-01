package stringobj

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	t.Parallel()

	type put struct {
		key   string
		value interface{}
	}

	tests := []struct {
		desc string
		puts []put
		want string
	}{
		{
			desc: "empty",
			want: "{}",
		},
		{
			desc: "non-empty",
			puts: []put{
				{"string", "bar"},
				{"int", 42},
				{"list", []string{}},
			},
			want: `{int: 42, list: [], string: bar}`,
		},
		{
			desc: "skip zero",
			puts: []put{
				{"string", ""},
				{"int", 0},
				{"list", nil},
			},
			want: "{}",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			var b Builder
			for _, i := range tt.puts {
				b.Put(i.key, i.value)
			}

			assert.Equal(t, tt.want, b.String())
		})
	}
}
