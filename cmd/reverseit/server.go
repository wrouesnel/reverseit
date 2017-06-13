package main

import (
	"context"
	"io"
	"net"
	"os"

	"github.com/hashicorp/yamux"
	"go.uber.org/zap"
)

// server is run on the side receiving the connection
func server(ctx context.Context, stdio io.ReadWriteCloser, log *zap.Logger, network string, listenAddr string) {
	// Setup the mux session on stdin
	muxSession, merr := yamux.Client(stdio, nil)
	if merr != nil {
		log.Fatal("Could not setup mux session:", zap.Error(merr))
	}

	// Start listening for connections on the TCP port
	l, lerr := net.Listen(network, listenAddr)
	if lerr != nil {
		log.Fatal("Could not start listening:", zap.String("listen_addr", listenAddr), zap.Error(lerr))
	}

	loopCtx, loopCancel := context.WithCancel(ctx)

	go func() {
		for {
			conn, aerr := l.Accept()
			if aerr != nil {
				log.Error("Error accepting connection:", zap.Error(aerr))
				loopCancel()
				return
			}
			connLog := log.With(
				zap.String("remote_addr", conn.RemoteAddr().String()), zap.String("local_addr", conn.LocalAddr().String()))
			stream, oerr := muxSession.Open()
			connLog.Debug("Proxying")
			if oerr != nil {
				log.Error("Error opening new mux stream:", zap.Error(oerr))
			} else {
				go handleProxy(connLog, conn, stream)
			}
		}
	}()
	<-loopCtx.Done()

	if cerr := l.Close(); cerr != nil {
		log.Error("Error while closing TCP listener", zap.Error(cerr))
	}

	if network == "unix" {
		if st, err := os.Stat(listenAddr); err == nil {
			if !st.IsDir() {
				// Clean up the socket file
				if err := os.Remove(listenAddr); err != nil {
					log.Error("Error cleaning up Unix socket file", zap.Error(err))
				}
			}
		}
	}
}
