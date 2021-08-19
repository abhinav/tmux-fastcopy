package paniclog

import (
	"bytes"
	"errors"
	"testing"

	"github.com/maxatome/go-testdeep/td"
)

func TestHandle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc string
		give interface{}

		wantMsg string // contains check
		wantErr string // equals check
	}{
		{desc: "nil"},
		{
			desc:    "string",
			give:    "foo",
			wantMsg: "panic: foo\n",
			wantErr: "foo",
		},
		{
			desc:    "error",
			give:    errors.New("great sadness"),
			wantMsg: "panic: great sadness\n",
			wantErr: "great sadness",
		},
		{
			desc:    "int",
			give:    42,
			wantMsg: "panic: 42",
			wantErr: "panic: 42",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.desc, func(t *testing.T) {
			t.Parallel()

			var buff bytes.Buffer
			got := Handle(tt.give, &buff)
			td.CmpContains(t, buff.String(), tt.wantMsg)

			if len(tt.wantErr) == 0 {
				td.CmpNoError(t, got)
			} else {
				td.CmpError(t, got)
				td.Cmp(t, got.Error(), tt.wantErr)
			}
		})
	}
}

func TestRecover(t *testing.T) {
	t.Parallel()

	t.Run("panic", func(t *testing.T) {
		t.Parallel()

		var (
			err  error
			buff bytes.Buffer
		)
		defer func() {
			td.CmpError(t, err)
			td.Cmp(t, err.Error(), "great sadness")
			td.CmpContains(t, buff.String(), "panic: great sadness\n")
		}()

		defer Recover(&err, &buff)

		panic("great sadness")
	})

	t.Run("no panic", func(t *testing.T) {
		t.Parallel()

		var (
			err  error
			buff bytes.Buffer
		)
		defer func() {
			td.CmpNoError(t, err)
			td.CmpEmpty(t, buff.String())
		}()

		defer Recover(&err, &buff)
	})
}
