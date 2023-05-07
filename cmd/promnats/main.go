package main

import (
	"flag"
	"fmt"
	"os/signal"
	"syscall"

	"os"
	"time"

	"github.com/kmpm/flagenvfile.go"
	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type options struct {
	Context   string
	Server    string
	Nkey      string
	Timeout   time.Duration
	Verbosity string
	// Portmap     PortMap
	MappingFile string
	PrettyLog   bool

	// DiscoveryHost string
	Version bool
	Host    string
}

var opts *options
var appVersion = "0.0.0-dev"

func init() {
	// initialize opts
	opts = &options{}

	// add some flags for connection
	flag.String("context", "", "context")
	flag.String("server", "", "server like "+nats.DefaultURL)
	flag.String("nkey", "", "pathto nkey file")

	// flags for other config
	flag.String("verbosity", "info", "debug|info|warn|error")
	flag.Duration("timeout", time.Second*2, "time waiting for replies")
	flag.String("mapping", "./mapping.txt", "path to file with <port>:<subject> mappings instead of args")
	flag.Bool("pretty", false, "pretty logging")

	// flag.StringVar(&opts.DiscoveryHost, "discovery", "", "ip / hostname to show in http_sd")
	flag.Bool("version", false, "show version and eit")
	flag.String("host", "", "host to use for http_sd. defaults to local IP if only 1")
}

func (o *options) fromFEF() error {
	opts.Verbosity = flagenvfile.GetString("verbosity")
	opts.PrettyLog = flagenvfile.GetBool("pretty")
	opts.Version = flagenvfile.GetBool("version")
	opts.MappingFile = flagenvfile.GetString("mapping")
	opts.Context = flagenvfile.GetString("context")
	opts.Server = flagenvfile.GetString("server")
	opts.Nkey = flagenvfile.GetString("nkey")
	opts.Host = flagenvfile.GetString("host")
	opts.Timeout = flagenvfile.GetDuration("timeout")
	return nil
}

func main() {
	flag.Parse()
	flagenvfile.BindFlagset(flag.CommandLine)
	flagenvfile.SetEnvPrefix("PN")
	opts.fromFEF()
	if opts.Version {
		fmt.Println(appVersion)
		os.Exit(0)
	}

	// setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	switch opts.Verbosity {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if opts.PrettyLog {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}

	log.Info().Str("version", appVersion).Msg("")
	log.Debug().Any("opts", opts).Msg("current options")
	if opts.Host == "" {
		ips := GetLocalIP()
		if len(ips) > 1 && opts.Host == "" {
			log.Fatal().
				Err(fmt.Errorf("more than 1 local ip found. please provide --host")).
				Strs("ips", ips).
				Msg("error getting host")
		}
		opts.Host = ips[0]
	}
	app := newApp()
	var err error
	var pm PortMaps
	// load mappings from file
	if opts.MappingFile != "" {

		pm, err = fileMappings(opts.MappingFile)
		if err != nil {
			log.Fatal().Err(err).Str("mapping", opts.MappingFile).Msg("error reading mappings")
		}

	}

	// get other mappings from args
	err = argMappings(pm)
	if err != nil {
		log.Fatal().Err(err).Msg("error parsing arguments")
	}

	// open connection by context or server
	if opts.Server != "" && opts.Context != "" {
		log.Fatal().Msg("you must not use both context and server")
	}

	if opts.Server == "" {
		app.nc, err = natscontext.Connect(opts.Context)
		if err != nil {
			log.Fatal().Err(err).Msg("error connecting using nats context")
		}
	} else {
		app.nc, err = nats.Connect(opts.Server)
		if err != nil {
			log.Fatal().Err(err).Msg("error connecting to server")
		}
	}

	err = app.start(":8083", opts.Host, 9000)
	if err != nil {
		log.Fatal().Err(err).Msg("error starting")
	}

	if len(pm) > 0 {
		// do aktual work
		err = app.refresh(pm)
		if err != nil {
			log.Fatal().Err(err).Msg("error doing work")
		}
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
	log.Info().Msg("running")
	<-done
	log.Info().Msg("closing")
	app.stop()
	log.Info().Msg("closed")
}

// argMappings adds arguments to anything that is allready in portmap
// if portmap is empty there must be att least 1 argument with portmap config
func argMappings(ps PortMaps) error {
	// if len(opts.Portmap) == 0 && flag.NArg() < 1 {
	// 	return fmt.Errorf("You must provide at least one <port>:<subject> argument")
	// }

	for _, pm := range flag.Args() {
		port, id, err := parsePortmap(pm)
		if err != nil {
			return err
		}
		ps[port] = id
	}
	return nil
}
