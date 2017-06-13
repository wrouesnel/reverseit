package main

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
)

// version is set by the makefile.
var version = "0.0.0-dev"

var CLI struct {
	Version   kong.VersionFlag `help:"Show version number"`
	LogLevel  string           `help:"Logging Level" enum:"debug,info,warning,error" default:"info"`
	LogFormat string           `help:"Logging format" enum:"console,json" default:"console"`
	Server    struct {
		Network    string `help:"Type of listener to create" enum:"unix,tcp" default:"tcp"`
		ListenAddr string `arg:"" help:"Address and port to listen on locally and forward to stdin/stdout"`
	} `cmd:""`

	Client struct {
		TargetAddr string   `arg:"" help:"Local port to reverse tunnels from the server too"`
		Command    []string `arg:"" help:"Command to establish the tunnel connection"`
	} `cmd:""`
}

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	ctx, cancelFn := context.WithCancel(context.Background())
	go func() {
		sig := <-sigCh
		zap.L().Info("Caught signal - exiting", zap.String("signal", sig.String()))
		cancelFn()
	}()

	mainConfig := &MainConfig{
		Ctx:  ctx,
		Args: os.Args[1:],
	}

	exitValue := realMain(mainConfig)
	os.Exit(exitValue)
}

type MainConfig struct {
	Ctx  context.Context
	Args []string
}

func configureLogging(logLevel string, logFormat string) *zap.Logger {
	// Configure logging
	config := zap.NewProductionConfig()
	config.Encoding = CLI.LogFormat
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(CLI.LogLevel)); err != nil {
		panic(err)
	}
	config.Level = zap.NewAtomicLevelAt(level)

	log, err := config.Build()
	if err != nil {
		panic(err)
	}

	return log
}

func realMain(mainConfig *MainConfig) int {
	vars := kong.Vars{}
	vars["version"] = version
	kongParser, err := kong.New(&CLI, vars)
	if err != nil {
		panic(err)
	}

	kongCtx, err := kongParser.Parse(mainConfig.Args)
	kongParser.FatalIfErrorf(err)

	log := configureLogging(CLI.LogLevel, CLI.LogFormat)

	stdioCtx, stdioClose := context.WithCancel(mainConfig.Ctx)

	stdio := newReadWriteCloser(os.Stdin, os.Stdout, func() error {
		log.Info("Close called on stdio")
		stdioClose()
		return nil
	})

	log.Debug("Logging configured",
		zap.String("log-format", CLI.LogFormat),
		zap.String("log-level", CLI.LogLevel))

	log.Debug("Mode", zap.String("mode", kongCtx.Command()))
	switch kongCtx.Command() {
	case "server <listen-addr>":
		server(stdioCtx, stdio, log.With(zap.String("mode", "server")), CLI.Server.Network, CLI.Server.ListenAddr)
	case "client <target-addr> <command>":
		client(stdioCtx, log.With(zap.String("mode", "client")), CLI.Client.TargetAddr, CLI.Client.Command)
	}

	log.Info("Exiting normally")
	return 0
}
