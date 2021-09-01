package paniclog

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Contains(t, buff.String(), tt.wantMsg)

			if len(tt.wantErr) == 0 {
				assert.NoError(t, got)
			} else {
				assert.Error(t, got)
				assert.Equal(t, tt.wantErr, got.Error())
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
			assert.Error(t, err)
			assert.Equal(t, "great sadness", err.Error())
			assert.Contains(t, buff.String(), "panic: great sadness\n")
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
			require.NoError(t, err)
			assert.Empty(t, buff.String())
		}()

		defer Recover(&err, &buff)
	})

	t.Run("no panic with error", func(t *testing.T) {
		t.Parallel()

		err := errors.New("great sadness")
		var buff bytes.Buffer

		defer func() {
			require.Error(t, err)
			assert.Contains(t, err.Error(), "great sadness")
			assert.Empty(t, buff.String())
		}()

		defer Recover(&err, &buff)
	})
}
