package main

import (
	"bytes"
	"io"
)

func newBufCloser(w io.Writer) io.WriteCloser {
	return &bufCloser{
		buffer: bytes.NewBuffer(nil),
		w:      w,
	}
}

// bufCloser buffers all output that is written to it until it's closed.
type bufCloser struct {
	buffer *bytes.Buffer
	w      io.Writer
}

func (w *bufCloser) Write(p []byte) (int, error) {
	return w.buffer.Write(p)
}

func (w *bufCloser) Close() error {
	_, err := w.buffer.WriteTo(w.w)
	return err
}
