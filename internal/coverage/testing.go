package coverage

import (
	"sync"
	"testing"
	_ "unsafe" // for go:linkname
)

//go:linkname testingCoverReport testing.coverReport
func testingCoverReport()

//go:linkname _testingCover testing.cover
var _testingCover testing.Cover

var _testingCoverMu sync.Mutex

// withCoverdata allows safely modifying the coverage data for this process.
func withCoverdata(f func(cover *testing.Cover)) {
	_testingCoverMu.Lock()
	defer _testingCoverMu.Unlock()

	f(&_testingCover)
}
