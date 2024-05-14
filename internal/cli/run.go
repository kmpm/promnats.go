package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/kmpm/flagenvfile.go"
	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
)

var programLevel = new(slog.LevelVar)
var appVersion = "0.0.0-dev"

func check(err error) {
	if err != nil {
		slog.Error("unexpected error", "error", err)
		panic(err)
	}
}

func logging(opts *Options) {
	// setup logging
	ho := &slog.HandlerOptions{Level: programLevel, AddSource: flagenvfile.GetBool("pretty")}

	h := slog.NewTextHandler(os.Stderr, ho)
	slog.SetDefault(slog.New(h))

	switch opts.Verbosity {
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
}

func newAppFromFlags() *App {
	// add some flags for connection

	opts, err := newOptionsFromFEF()
	if err != nil {
		slog.Error("error getting options", "error", err)
		os.Exit(1)
	}
	// if opts.Version {
	// 	fmt.Println(appVersion)
	// 	os.Exit(0)
	// }
	return newAppFromOptions(opts)
}

func newConnectionFromOptions(opts *Options) (*nats.Conn, error) {
	var nc *nats.Conn
	var err error
	appname := "promnats " + appVersion
	// open connection by context or server
	if opts.Server != "" && opts.Context != "" {
		return nil, errors.New("you must not use both context and server")

	}

	nopts := []nats.Option{
		nats.Name(appname),
		nats.ErrorHandler(func(_ *nats.Conn, sub *nats.Subscription, err error) {
			slog.Error("error", "sub", sub.Subject, "error", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			slog.Info("reconnected")
		}),
		nats.ConnectHandler(func(nc *nats.Conn) {
			slog.Info("connected", "servers", nc.Servers())
		}),
	}

	if opts.Server == "" {
		nc, err = natscontext.Connect(opts.Context, nopts...)
		if err != nil {
			return nil, fmt.Errorf("error connecting using nats context: %w", err)
		}
	} else {
		nc, err = nats.Connect(opts.Server, nopts...)
		if err != nil {
			return nil, fmt.Errorf("error connecting to server: %w", err)
		}
	}
	return nc, nil
}

func newAppFromOptions(opts *Options) *App {
	logging(opts)
	var err error
	var portS string
	_, portS, err = net.SplitHostPort(opts.Address)
	check(err)
	check(err)
	port, err := strconv.Atoi(portS)
	check(err)

	if opts.Host == "" {
		ips := GetLocalIP()
		if len(ips) > 1 && opts.Host == "" {
			slog.Error("error! more than 1 local ip found. please provide --host", "ips", ips)
			os.Exit(1)
			// log.Fatal().
			// 	Err(fmt.Errorf("more than 1 local ip found. please provide --host")).
			// 	Strs("ips", ips).
			// 	Msg("error getting host")
		}
		opts.Host = ips[0]
	}
	nc, err := newConnectionFromOptions(opts)
	check(err)

	app := newApp(opts.Timeout, opts.Host, opts.Address, port)
	app.nc = nc
	return app
}

func Run(context.Context) {
	//TODO: configure the app
	app := newAppFromFlags()
	app.start()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		// fmt.Println()
		// fmt.Println(sig)
		done <- true
	}()

	// wait for things to close
	// slog.Info("running")
	<-done
	slog.Info("stopping")
	app.stop()
	slog.Info("stopped")

}
