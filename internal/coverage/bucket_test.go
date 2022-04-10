package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucketNoop(t *testing.T) {
	b, err := NewBucket("")
	require.NoError(t, err)

	assert.Empty(t, b.Dir(), "directory should be empty")
	assert.NoError(t, b.Finalize(), "finalize should succeed")
}

func TestBucketMerging(t *testing.T) {
	bucket, err := NewBucket("count")
	require.NoError(t, err)

	var cover testing.Cover
	bucket.withCoverdata = func(f func(*testing.Cover)) {
		f(&cover)
	}

	coverFiles := [][]string{
		{
			"mode: count",
			"example.com/foo.go:12.34,56.78 2 7",
		},
		{
			"mode: count",
			"example.com/foo.go:12.34,56.78 2 9",
		},
	}

	for i, lines := range coverFiles {
		body := strings.Join(lines, "\n") + "\n"
		fname := filepath.Join(bucket.Dir(), fmt.Sprintf("cover%d", i))
		require.NoError(t, os.WriteFile(fname, []byte(body), 0644))
	}

	require.NoError(t, bucket.Finalize())

	assert.Equal(t, uint32(16), cover.Counters["example.com/foo.go"][0])
}
