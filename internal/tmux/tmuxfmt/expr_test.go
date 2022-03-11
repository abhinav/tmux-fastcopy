package tmuxfmt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBinaryOp_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give BinaryOp
		want string
	}{
		{"equals", Equals, "=="},
		{"not equals", NotEquals, "!="},
		{"less than", LessThan, "<"},
		{"greater than", GreaterThan, ">"},
		{"less than equals", LessThanEquals, "<="},
		{"greater than equals", GreaterThanEquals, ">="},
		{"unrecognized", BinaryOp(-1), "BinaryOp(-1)"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, tt.give.String())
		})
	}
}
