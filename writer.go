package main

import (
	"fmt"
	"io"
)

// newNameWriter returns an io.Writer that prepends the given name
func newNameWriter(name string, w io.Writer) io.Writer {
	return &nameWriter{
		name: name,
		w:    w,
	}
}

// nameWriter prepends a name in [] to each write
type nameWriter struct {
	name string
	w    io.Writer
}

func (n *nameWriter) Write(p []byte) (int, error) {
	l := len(p)
	_, err := fmt.Fprintf(n.w, "[%s] %s", n.name, p)
	return l, err
}
