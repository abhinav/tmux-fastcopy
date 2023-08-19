package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
)

type handler struct {
	W     io.Writer
	Level Level

	attrs []byte
	group []byte
}

var _ slog.Handler = (*handler)(nil)

func (h *handler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return lvl >= h.Level
}

var (
	_reset = []byte("\x1b[0m")
	_bold  = []byte("\x1b[1m")
	_dim   = []byte("\x1b[2m")

	_boldDim          = []byte("\x1b[2;1m")
	_brightBoldRed    = []byte("\x1b[91;1m")
	_brightBoldYellow = []byte("\x1b[93;1m")
	_brightBoldGreen  = []byte("\x1b[92;1m")
)

func (h *handler) Handle(ctx context.Context, rec slog.Record) error {
	buf := *getBuf()
	defer putBuf(&buf)

	lvl, err := rec.Level.MarshalText()
	if err != nil {
		return err
	}

	if rec.Level >= slog.LevelError {
		buf = append(buf, _brightBoldRed...)
	} else if rec.Level >= slog.LevelWarn {
		buf = append(buf, _brightBoldYellow...)
	} else if rec.Level >= slog.LevelInfo {
		buf = append(buf, _brightBoldGreen...)
	} else {
		buf = append(buf, _boldDim...)
	}
	buf = append(buf, lvl...)
	buf = append(buf, _reset...)
	buf = append(buf, ' ')

	buf = append(buf, _bold...)
	buf = append(buf, rec.Message...)
	buf = append(buf, _reset...)

	if len(h.attrs) > 0 {
		buf = append(buf, ' ')
		buf = append(buf, h.attrs...)
	}

	rec.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, h.group, a)
		return true
	})

	buf = append(buf, '\n')
	_, err = h.W.Write(buf)
	return err
}

func (h *handler) appendAttr(buf []byte, group []byte, a slog.Attr) []byte {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf
	}

	if a.Value.Kind() == slog.KindGroup {
		group := group
		if len(group) > 0 {
			group = append(group, '.')
		}
		group = append(group, a.Key...)
		for _, a := range a.Value.Group() {
			buf = h.appendAttr(buf, group, a)
		}

		return buf
	}

	if len(buf) > 0 && buf[len(buf)-1] != ' ' {
		buf = append(buf, ' ')
	}

	buf = append(buf, _dim...)
	if len(group) > 0 {
		buf = append(buf, group...)
		buf = append(buf, '.')
	}
	buf = append(buf, a.Key...)
	buf = append(buf, '=')
	buf = append(buf, _reset...)

	switch a.Value.Kind() {
	case slog.KindString:
		if strings.ContainsAny(a.Value.String(), ` \"=`) {
			// TODO: check for non-printable characters
			buf = append(buf, strconv.Quote(a.Value.String())...)
		} else {
			buf = append(buf, a.Value.String()...)
		}

	case slog.KindInt64:
		buf = strconv.AppendInt(buf, a.Value.Int64(), 10)

	case slog.KindUint64:
		buf = strconv.AppendUint(buf, a.Value.Uint64(), 10)

	case slog.KindFloat64:
		buf = strconv.AppendFloat(buf, a.Value.Float64(), 'f', -1, 64)

	case slog.KindBool:
		buf = strconv.AppendBool(buf, a.Value.Bool())

	case slog.KindDuration:
		buf = append(buf, a.Value.Duration().String()...)

	case slog.KindTime:
		buf = append(buf, a.Value.Time().String()...)

	case slog.KindAny:
		buf = fmt.Appendf(buf, "%v", a.Value.Any())
	}

	return buf
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	out := *h
	if len(out.attrs) > 0 {
		out.attrs = append(out.attrs, ' ')
	}
	for i, a := range attrs {
		if i > 0 {
			out.attrs = append(out.attrs, ' ')
		}
		out.attrs = out.appendAttr(out.attrs, h.group, a)
	}
	return &out
}

func (h *handler) WithGroup(name string) slog.Handler {
	out := *h
	if len(out.group) > 0 {
		out.group = append(out.group, '.')
	}
	out.group = append(out.group, name...)
	return &out
}

var _bufPool = sync.Pool{
	New: func() interface{} {
		bs := make([]byte, 0, 1024)
		return &bs
	},
}

func getBuf() *[]byte {
	return _bufPool.Get().(*[]byte)
}

func putBuf(bs *[]byte) {
	*bs = (*bs)[:0]
	_bufPool.Put(bs)
}
