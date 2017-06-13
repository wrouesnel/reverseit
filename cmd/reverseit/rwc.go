package main

import "io"

type readWriteCloser struct {
	io.Reader
	io.Writer
	closeFn func() error
}

func (sc *readWriteCloser) Close() error {
	return sc.closeFn()
}

func newReadWriteCloser(r io.Reader, w io.Writer, closeFn func() error) io.ReadWriteCloser {
	return &readWriteCloser{r, w, closeFn}
}
