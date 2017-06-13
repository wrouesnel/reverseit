package main

import (
	"bufio"
	"context"
	"io"
	"net"
	"os/exec"

	"github.com/hashicorp/yamux"
	"go.uber.org/zap"
)

// client is run on the side establishing the connection.
func client(ctx context.Context, log *zap.Logger,
	targetAddr string, command []string) {

	commandLog := log.With(zap.Strings("command", command))

	realCmd, err := exec.LookPath(command[0])
	if err != nil {
		log.Fatal("Could not find the command to launch", zap.Error(err))
	}
	commandLog.Info("Launching IO command", zap.String("executable", realCmd))

	// Execute the subcommand
	processCtx, processCancelFn := context.WithCancel(ctx)

	cmd := exec.CommandContext(processCtx, realCmd, command[1:]...)
	stdOutReader, err := cmd.StdoutPipe()
	if err != nil {
		commandLog.Fatal("Error creating pipe for process", zap.Error(err))
	}
	stdInWriter, err := cmd.StdinPipe()
	if err != nil {
		commandLog.Fatal("Error creating pipe for process", zap.Error(err))
	}
	stdErrReader, err := cmd.StderrPipe()
	if err != nil {
		commandLog.Fatal("Error creating pipe for process", zap.Error(err))
	}
	bioReader := bufio.NewReader(stdErrReader)
	go func() {
		for {
			stderrLine, err := bioReader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					log.Info("EOF from standard error")
					return
				} else {
					log.Error("Error from standard error pipe")
					return
				}
			}
			log.Debug(stderrLine, zap.String("pipe", "stderr"))
		}
	}()

	processPipe := newReadWriteCloser(stdOutReader, stdInWriter, func() error {
		log.Info("Cancelling process context due to pipe close")
		processCancelFn()
		return nil
	})

	if err := cmd.Start(); err != nil {
		commandLog.Fatal("Starting command failed")
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			commandLog.Error("Error from command exit", zap.Error(err))
		}
		processCancelFn()
	}()

	// Setup the mux server on the process stdin/stdout
	ml, merr := yamux.Server(processPipe, nil)
	if merr != nil {
		log.Fatal("Could not setup mux session:", zap.Error(merr))
	}

	loopCtx, loopCancel := context.WithCancel(processCtx)

	go func() {
		for {
			stream, aerr := ml.Accept()
			if aerr != nil {
				log.Error("Error accepting connection on mux:", zap.Error(aerr))
				loopCancel()
				return
			}
			connLog := log.With(
				zap.String("remote_addr", stream.RemoteAddr().String()),
				zap.String("local_addr", stream.LocalAddr().String()))

			connLog.Debug("Accepting connection on mux")
			conn, oerr := net.Dial("tcp", targetAddr)

			proxyLog := log.With(
				zap.String("target_addr", targetAddr))

			connLog.Debug("Proxying")
			if oerr != nil {
				proxyLog.Error("Error opening connection to target:", zap.Error(oerr))
			} else {
				go handleProxy(proxyLog, stream, conn)
			}
		}
	}()
	<-loopCtx.Done()

	// Closing the listener will kill the process by the mux
	if cerr := ml.Close(); cerr != nil {
		log.Error("Error while closing Mux listener:", zap.Error(cerr))
	}
}
