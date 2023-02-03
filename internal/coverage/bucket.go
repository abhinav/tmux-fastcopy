package coverage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/multierr"
	"golang.org/x/tools/cover"
)

// Bucket is a collection of coverage profiles obtained from calling an
// external program.
//
// We use this in conjunction with the Report function to get coverage data
// from a copy of tmux-fastcopy spawned by tmux.
//
// Roughly, the ingtegration test uses the usual TestMain hijacking method to
// allow control of the binary spawned by tmux. When running in coverage mode,
//
//   - the test sets up a bucket to place coverage data in, and communicates the
//     path to this bucket to the spawned binary with an environment variable
//   - the spanwed binary, if this environment variable is set, generates a
//     coverage report into this directory using coverage.Report
//   - afterwards, the test uses Bucket.Finalize to merge coverage data from the
//     spawned binary back into the current process
//
// This is inspired by [go-internal/testscript][1], but instead of replaying
// the full test machinery, we just invoke or modify a couple private
// functions/variables in the testing packge to make this work.
//
// [1]: https://github.com/rogpeppe/go-internal/blob/3461ca1f2345421c6d6d05407a3ac0381bbd5c42/testscript/cover.go
type Bucket struct {
	dir       string
	coverMode string
	isCount   bool

	withCoverdata func(func(*testing.Cover))
}

// NewBucket builds a new coverage bucket with the given coverage mode.
// Returns a no-op bucket if we're not running with coverage.
//
// This should only be called from inside a test, or *after* flag.Parse in
// TestMain.
func NewBucket(coverMode string) (*Bucket, error) {
	if len(coverMode) == 0 {
		return nil, nil
	}

	dir, err := os.MkdirTemp("", "fastcopy-coverage-*")
	if err != nil {
		return nil, fmt.Errorf("create coverage directory: %w", err)
	}

	return &Bucket{
		dir:           dir,
		coverMode:     coverMode,
		isCount:       coverMode == "count",
		withCoverdata: withCoverdata,
	}, nil
}

// Dir reports the directory inside which coverage data should be placed.
//
// Coverage-instrumented binaries should use coverage.Report to place their
// coverage data inside this directory.
func (b *Bucket) Dir() string {
	if b == nil {
		return ""
	}
	return b.dir
}

// Finalize cleans up temporary resources allocated by this bucket and merges
// coverage data from the external coverage-instrumented binaries back into
// this process.
func (b *Bucket) Finalize() (err error) {
	if b == nil {
		return nil
	}

	defer func() {
		err = multierr.Append(err, os.RemoveAll(b.dir))
	}()

	return fs.WalkDir(os.DirFS(b.dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.Type().IsRegular() {
			return nil
		}

		return b.mergeProfiles(filepath.Join(b.dir, path))
	})
}

func (b *Bucket) mergeProfiles(path string) error {
	profiles, err := cover.ParseProfiles(path)
	if err != nil {
		return err
	}

	for _, p := range profiles {
		if p.Mode != b.coverMode {
			return fmt.Errorf("%v:unexpected coverage mode %q, expected %q", p.FileName, p.Mode, b.coverMode)
		}

		b.withCoverdata(func(cover *testing.Cover) {
			b.mergeProfile(cover, p)
		})
	}

	return nil
}

func (b *Bucket) mergeProfile(cover *testing.Cover, p *cover.Profile) {
	if cover.Counters == nil {
		cover.Counters = make(map[string][]uint32)
	}
	if cover.Blocks == nil {
		cover.Blocks = make(map[string][]testing.CoverBlock)
	}

	counters := cover.Counters[p.FileName]
	blocks := cover.Blocks[p.FileName]

	for i, blk := range p.Blocks {
		if i >= len(counters) {
			counters = append(counters, uint32(blk.Count))
			blocks = append(blocks, testing.CoverBlock{
				Line0: uint32(blk.StartLine),
				Col0:  uint16(blk.StartCol),
				Line1: uint32(blk.EndLine),
				Col1:  uint16(blk.EndCol),
				Stmts: uint16(blk.NumStmt),
			})
			continue
		}

		if b.isCount {
			counters[i] += uint32(blk.Count)
		} else {
			counters[i] |= uint32(blk.Count)
		}
	}

	cover.Counters[p.FileName] = counters
	cover.Blocks[p.FileName] = blocks
}
