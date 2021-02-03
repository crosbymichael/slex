package main

import (
	"bytes"
	"io"
	"time"
)

func newWriter(j *job) io.Writer {
	return &writer{
		j: j,
	}
}

// writer buffers all output that is written to it until it's closed.
type writer struct {
	j *job
}

func (w *writer) Write(p []byte) (int, error) {
	lines := bytes.Split(p, []byte("\n"))
	for _, l := range lines {
		w.j.lines = append(w.j.lines, string(l))
		w.j.signal <- struct{}{}
		time.Sleep(50 * time.Millisecond)
	}
	//w.j.signal <- struct{}{}
	return len(p), nil
}
