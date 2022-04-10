package coverage

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/abhinav/tmux-fastcopy/internal/stub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportNotATest(t *testing.T) {
	t.Cleanup(stub.Replace(
		&_flagSet,
		flag.NewFlagSet("foo", flag.ContinueOnError),
	))

	err := Report("foo")
	require.Error(t, err, "report should fail")
	assert.Contains(t, err.Error(), "not inside a test")
}

func TestReport(t *testing.T) {
	out := filepath.Join(t.TempDir(), "new_cover.out")
	require.NoError(t, Report(out))

	_, err := os.Stat(out)
	require.NoError(t, err, "file should exist")
}
