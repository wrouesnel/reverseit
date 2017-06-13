package main

import (
	"github.com/hashicorp/yamux"
	"github.com/wrouesnel/go.log"

	"flag"
	"gopkg.in/alecthomas/kingpin.v2"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
)

// Version is set by the makefile.
var Version = "0.0.0-dev"

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	app := kingpin.New("reverseit", "Simple TCP reverse proxy tool")
	app.Version(Version)

	loglevel := app.Flag("log-level", "Logging Level").Default("info").String()
	logformat := app.Flag("log-format", "If set use a syslog logger or JSON logging. Example: logger:syslog?appname=bob&local=7 or logger:stdout?json=true. Defaults to stderr.").Default("stderr").String()

	clientCmd := app.Command("client", "Start in client mode (connects stdio mux to local TCP port")
	targetAddr := clientCmd.Arg("target", "destination address for incoming TCP connections in host:port format").String()

	serverCmd := app.Command("server", "Start in server mode (listens to TCP port and sends over stdio")
	listenAddr := serverCmd.Arg("listen-port", "port to listen for connections on in host:port format (specify as :port to bind to all addresses)").String()

	stdio := newReadWriteCloser(os.Stdin, os.Stdout, func() error {
		log.Infoln("Close called on stdio")
		return nil
	})

	cmd := kingpin.MustParse(app.Parse(os.Args[1:]))

	if err := flag.Set("log.level", *loglevel); err != nil {
		log.Fatalln("Could not set log.level flag")
	}
	if err := flag.Set("log.format", *logformat); err != nil {
		log.Fatalln("Could not set log.format flag")
	}

	log.Debugln("Log Level:", *loglevel)

	switch cmd {
	case clientCmd.FullCommand():
		client(stdio, sigCh, log.With("mode", "client"), *targetAddr)
	case serverCmd.FullCommand():
		server(stdio, sigCh, log.With("mode", "server"), *listenAddr)
	}
}

// client is run on the side establishing the connection.
func client(stdio io.ReadWriteCloser, exitCh <-chan os.Signal, log log.Logger, targetAddr string) {
	// Setup the mux server on stdin
	ml, merr := yamux.Server(stdio, nil)
	if merr != nil {
		log.Fatalln("Could not setup mux session:", merr)
	}

	go func() {
		for {
			stream, aerr := ml.Accept()
			if aerr != nil {
				log.Errorln("Error accepting connection on mux:", aerr)
				return
			}
			log.Debugln("Accepting connection on mux:", stream.RemoteAddr(), "->", stream.LocalAddr())
			conn, oerr := net.Dial("tcp", targetAddr)
			log.Debugln("Proxying to:", conn.LocalAddr(), "->", conn.RemoteAddr())
			if oerr != nil {
				log.Errorln("Error opening connection to target:", targetAddr, oerr)
			} else {
				go handleProxy(log, stream, conn)
			}
		}
	}()

	<-exitCh
	if cerr := ml.Close(); cerr != nil {
		log.Errorln("Error while closing Mux listener:", cerr)
	}
}

// server is run on the side receiving the connection
func server(stdio io.ReadWriteCloser, exitCh <-chan os.Signal, log log.Logger, listenAddr string) {
	// Setup the mux session on stdin
	muxSession, merr := yamux.Client(stdio, nil)
	if merr != nil {
		log.Fatalln("Could not setup mux session:", merr)
	}

	// Start listening for connections on the TCP port
	l, lerr := net.Listen("tcp", listenAddr)
	if lerr != nil {
		log.Fatalln("Could not start listening:", listenAddr, lerr)
	}

	go func() {
		for {
			conn, aerr := l.Accept()
			if aerr != nil {
				log.Errorln("Error accepting connection:", aerr)
				return
			}
			log.Debugln("Accepting connection on port:", conn.RemoteAddr(), "->", conn.LocalAddr())
			stream, oerr := muxSession.Open()
			log.Debugln("Proxying to:", stream.LocalAddr(), "->", stream.RemoteAddr())
			if oerr != nil {
				log.Errorln("Error opening new mux stream:", oerr)
			} else {
				go handleProxy(log, conn, stream)
			}
		}
	}()

	<-exitCh
	if cerr := l.Close(); cerr != nil {
		log.Errorln("Error while closing TCP listener:", cerr)
	}
}

func handleProxy(log log.Logger, incoming, outgoing io.ReadWriteCloser) {
	defer func() { logErr(incoming.Close()) }()
	defer func() { logErr(outgoing.Close()) }()
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
			log.Debugln("All connections finished")
			break
		}
	}
	log.Debugln("Proxy session finished")
}

func pipe(log log.Logger, src io.Reader, dst io.Writer, closeCh chan struct{}) {
	data := make([]byte, 4096)
	for {
		readBytes, rerr := src.Read(data)
		if rerr != nil {
			if rerr != io.EOF {
				log.Errorln("read error:", rerr)
			}
			close(closeCh)
			return
		}
		writtenBytes, werr := dst.Write(data[:readBytes])
		if werr != nil {
			if werr != io.EOF {
				log.Errorln("write error:", werr)
			}
			close(closeCh)
			return
		}
		if writtenBytes != readBytes {
			log.Errorln("Failed to completely forward data")
			close(closeCh)
			return
		}
	}
}

func logErr(err error) {
	if err != nil {
		log.Errorln(err)
	}
}

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
