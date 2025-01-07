package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os/signal"
	"strconv"
	"syscall"

	"os"
	"time"

	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
)

type options struct {
	Version      bool
	LogVerbosity string
	LogPretty    bool
	LogFormat    string

	Context string
	Server  string
	Nkey    string
	Timeout time.Duration
	Address string
	Host    string
}

var opts *options
var appVersion = "0.0.0-dev"
var programLevel = new(slog.LevelVar)

func main() {
	// initialize opts
	opts = &options{}

	// flags for logging
	flag.StringVar(&opts.LogVerbosity, "verbosity", "info", "debug|info|warn|error")
	flag.BoolVar(&opts.LogPretty, "pretty", false, "pretty logging")
	flag.StringVar(&opts.LogFormat, "logformat", "text", "text|json")

	// add some flags for connection
	flag.StringVar(&opts.Context, "context", "", "<context name> to use for connection")
	flag.StringVar(&opts.Server, "server", nats.DefaultURL, "server like "+nats.DefaultURL)
	flag.StringVar(&opts.Nkey, "nkey", "", "path to nkey file")

	// flags for other config
	flag.DurationVar(&opts.Timeout, "timeout", time.Second*2, "time waiting for replies")

	flag.StringVar(&opts.Address, "address", ":8083", "address to listen on")
	flag.StringVar(&opts.Host, "host", "", "host to use for http_sd. defaults to local IP if only 1")
	// flags not in opts
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "show version and eit")

	flag.Parse()

	var err error
	err = mergeWithEnv(flag.CommandLine, "PN")
	check(err)

	if showVersion {
		fmt.Println(appVersion)
		os.Exit(0)
	}

	// setup logging
	ho := &slog.HandlerOptions{Level: programLevel, AddSource: opts.LogPretty}
	var h slog.Handler
	switch opts.LogFormat {
	case "json":
		h = slog.NewJSONHandler(os.Stderr, ho)
	case "text":
		h = slog.NewTextHandler(os.Stderr, ho)
	default:
		panic("unknown log format: " + opts.LogFormat)
	}

	slog.SetDefault(slog.New(h))

	switch opts.LogVerbosity {
	case "debug":
		programLevel.Set(slog.LevelDebug)
	case "info":
		programLevel.Set(slog.LevelInfo)
	case "warn":
		programLevel.Set(slog.LevelWarn)
	case "error":
		programLevel.Set(slog.LevelError)
	default:
		programLevel.Set(slog.LevelInfo)
	}

	// if opts.PrettyLog {
	// 	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	// }

	var portS string
	_, portS, err = net.SplitHostPort(opts.Address)
	check(err)
	port, err := strconv.Atoi(portS)
	check(err)

	slog.Info("starting promnats", "version", appVersion, "options", *opts)

	if opts.Host == "" {
		ips := GetLocalIP()
		if len(ips) > 1 && opts.Host == "" {
			slog.Error("error! more than 1 local ip found. please provide --host", "ips", ips)
			os.Exit(1)
		}
		opts.Host = ips[0]
	}
	app := newApp()

	appname := "promnats " + appVersion

	app.nc, err = connect(appname)
	if err != nil {
		slog.Error("error connecting to nats", "error", err)
		os.Exit(1)
	}

	err = app.start(opts.Address, opts.Host, port)
	if err != nil {
		slog.Error("error starting", "error", err)
		os.Exit(1)
	}

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	// wait for things to close
	slog.Info("running")
	<-done
	slog.Info("closing")
	app.stop()
	slog.Info("closed")
}

func connect(appname string) (nc *nats.Conn, err error) {
	nopts := []nats.Option{
		nats.Name(appname),
	}

	if opts.Context != "" {
		nc, err = natscontext.Connect(opts.Context, nopts...)
		if err != nil {
			return nil, err
		}
	} else {
		if opts.Nkey != "" {
			if o, err := nats.NkeyOptionFromSeed(opts.Nkey); err != nil {
				slog.Error("error creating nkey option", "error", err)
				os.Exit(1)
			} else {
				nopts = append(nopts, o)
			}
		}

		nc, err = nats.Connect(opts.Server, nopts...)
		if err != nil {
			return nil, err
		}
	}
	nc.SetClosedHandler(func(_ *nats.Conn) {
		slog.Warn("nats connection closed")
	})
	nc.SetDisconnectHandler(func(_ *nats.Conn) {
		slog.Warn("nats disconnected")
	})
	nc.SetReconnectHandler(func(ncx *nats.Conn) {
		slog.Warn("nats reconnected", "servers", ncx.Servers())
	})
	nc.SetErrorHandler(func(_ *nats.Conn, s *nats.Subscription, err error) {
		slog.Error("nats error", "error", err, "subject", s.Subject)
	})

	return nc, nil
}
