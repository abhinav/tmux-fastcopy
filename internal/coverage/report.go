package coverage

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/abhinav/tmux-fastcopy/internal/stub"
)

var (
	// We're changing globals here so this helps ensure there are no
	// concurrent calls overwriting each other.
	_mu sync.Mutex

	_flagSet = flag.CommandLine
)

// Report reports the coverage of the current binary to the given file.
//
// This operates by calling a private function in the testing package,
// so it's not guaranteed to continue working in the future.
func Report(path string) error {
	_mu.Lock()
	defer _mu.Unlock()

	f := _flagSet.Lookup("test.coverprofile")
	if f == nil {
		return errors.New("not inside a test binary")
	}

	// Restore the old value afterwards if we can access it.
	if g, ok := f.Value.(flag.Getter); ok {
		defer func(old string) {
			f.Value.Set(old)
		}(g.Get().(string))
	}

	if err := f.Value.Set(path); err != nil {
		return fmt.Errorf("set coverage destination: %w", err)
	}

	// testingCoverReport prints to stdout. Make it be quiet if we can.
	devNull, err := os.Open(os.DevNull)
	if err == nil {
		defer stub.Replace(&os.Stdout, devNull)()
	}

	testingCoverReport()
	return nil
}
