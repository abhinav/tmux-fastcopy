package stub

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplace(t *testing.T) {
	t.Parallel()

	value := 42
	restore := Replace(&value, 100)
	assert.Equal(t, 100, value)
	restore()
	assert.Equal(t, 42, value)
}
