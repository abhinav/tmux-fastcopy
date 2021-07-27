package log

import "bytes"

// Writer is an io.Writer that writes to the provided logger, splitting
// messages across newlines into new log entries.
type Writer struct {
	Log   *Logger
	Level Level

	buff bytes.Buffer
}

func (w *Writer) Write(bs []byte) (int, error) {
	n := len(bs)
	for len(bs) > 0 {
		bs = w.takeNextLine(bs)
	}
	return n, nil
}

func (w *Writer) takeNextLine(line []byte) (remaining []byte) {
	idx := bytes.IndexByte(line, '\n')
	if idx < 0 {
		// If there are no newlines, buffer the entire string.
		w.buff.Write(line)
		return nil
	}

	// Split on the newline, buffer and flush the left.
	line, remaining = line[:idx], line[idx+1:]

	// Fast path: if we don't have a partial message from a previous write
	// in the buffer, skip the buffer and log directly.
	if w.buff.Len() == 0 {
		w.logLine(line)
		return
	}

	w.buff.Write(line)

	// Log empty messages in the middle of the stream so that we don't lose
	// information when the user writes "foo\n\nbar".
	w.flush(true /* allowEmpty */)

	return remaining
}

// Close closes the Writer, flushing any buffered data to the underlying log.
func (w *Writer) Close() error {
	// Don't allow empty messages on Close because we don't want an
	// extraneous empty message at the end of the stream -- it's common for
	// files to end with a newline.
	w.flush(false /* allowEmpty */)
	return nil
}

func (w *Writer) flush(allowEmpty bool) {
	if allowEmpty || w.buff.Len() > 0 {
		w.logLine(w.buff.Bytes())
	}
	w.buff.Reset()
}

func (w *Writer) logLine(b []byte) {
	w.Log.Log(w.Level, "%s", b)
}
