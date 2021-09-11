package envtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetenv(t *testing.T) {
	t.Parallel()

	env := MustPairs(
		"FOO", "bar",
		"BAZ", "",
	)

	t.Run("match", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "bar", env.Getenv("FOO"))
	})

	t.Run("empty match", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, env.Getenv("BAZ"))
	})

	t.Run("empty no match", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, env.Getenv("QUX"))
	})
}

func TestGetenvNil(t *testing.T) {
	t.Parallel()

	var env *Env
	assert.Empty(t, env.Getenv("QUX"))
}

func TestMustPairsOddArguments(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		MustPairs("foo", "bar", "baz")
	})
}

func TestPairsOddArguments(t *testing.T) {
	t.Parallel()

	_, err := Pairs("foo", "bar", "baz")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not even")
}
