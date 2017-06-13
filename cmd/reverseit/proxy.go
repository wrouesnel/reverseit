package main

import (
	"io"

	"go.uber.org/zap"
)

func LogIfErr(err error) {
	if err != nil {
		zap.L().Error("Error", zap.Error(err))
	}
}

// handleProxy is spawned to copy data
func handleProxy(log *zap.Logger, incoming, outgoing io.ReadWriteCloser) {
	defer func() { LogIfErr(incoming.Close()) }()
	defer func() { LogIfErr(outgoing.Close()) }()
	// Forward data between connections
	closedSrcDest := make(chan struct{})
	closedDestSrc := make(chan struct{})
	go pipe(log, incoming, outgoing, closedSrcDest)
	go pipe(log, outgoing, incoming, closedDestSrc)
	for {
		select {
		case <-closedSrcDest:
			closedSrcDest = nil
		case <-closedDestSrc:
			closedDestSrc = nil
		}
		if closedDestSrc == nil && closedSrcDest == nil {
			log.Debug("All connections finished")
			break
		}
	}
	log.Debug("Proxy session finished")
}
