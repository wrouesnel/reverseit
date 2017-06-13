package main

import (
	"io"

	"go.uber.org/zap"
)

func pipe(log *zap.Logger, src io.Reader, dst io.Writer, closeCh chan struct{}) {
	data := make([]byte, 4096)
	for {
		readBytes, rerr := src.Read(data)
		if rerr != nil {
			if rerr != io.EOF {
				log.Error("read error", zap.Error(rerr))
			}
			close(closeCh)
			return
		}
		writtenBytes, werr := dst.Write(data[:readBytes])
		if werr != nil {
			if werr != io.EOF {
				log.Error("write error:", zap.Error(werr))
			}
			close(closeCh)
			return
		}
		if writtenBytes != readBytes {
			log.Error("Failed to completely forward data", zap.Int("written_bytes", writtenBytes), zap.Int("read_bytes", readBytes))
			close(closeCh)
			return
		}
	}
}
