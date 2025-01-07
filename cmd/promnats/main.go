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
	flag.StringVar(&opts.Context, "context", "", "context")
	flag.StringVar(&opts.Server, "server", "", "server like "+nats.DefaultURL)
	flag.StringVar(&opts.Nkey, "nkey", "", "pathto nkey file")

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
	// open connection by context or server
	if opts.Server != "" && opts.Context != "" {
		slog.Error("you must not use both context and server")
		os.Exit(1)
	}

	if opts.Server == "" {
		app.nc, err = natscontext.Connect(opts.Context, nats.Name(appname))
		if err != nil {
			slog.Error("error connecting using nats context", "error", err)
			os.Exit(1)
		}
	} else {
		app.nc, err = nats.Connect(opts.Server, nats.Name(appname))
		if err != nil {
			slog.Error("error connecting to server", "error", err)
			os.Exit(1)
		}
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
