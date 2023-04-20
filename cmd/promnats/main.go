package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"

	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Opts struct {
	Context     string
	Server      string
	Timeout     time.Duration
	Verbosity   string
	Portmap     map[int]string
	MappingFile string
}

var opts Opts

func init() {
	opts = Opts{
		Portmap: make(map[int]string),
	}
	flag.StringVar(&opts.Context, "context", "", "context")
	flag.StringVar(&opts.Server, "server", "", "server like "+nats.DefaultURL)
	flag.StringVar(&opts.MappingFile, "mapping", "", "path to file with <port>:<subject> mappings instead of args")
	flag.DurationVar(&opts.Timeout, "timeout", time.Second*2, "time waiting for replies")
	flag.StringVar(&opts.Verbosity, "verbosity", "info", "debug|info|warn|error")
}

func main() {
	flag.Parse()
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

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	var err error
	if opts.MappingFile != "" {
		err = fileMappings(opts.MappingFile)
		if err != nil {
			log.Fatal().Err(err).Str("mapping", opts.MappingFile).Msg("error reading mappings")
		}
	}

	err = argMappings()
	if err != nil {
		log.Fatal().Err(err).Msg("error parsing arguments")
	}

	var nc *nats.Conn

	if opts.Server == "" {
		nc, err = natscontext.Connect(opts.Context)
		if err != nil {
			// log.Fatalf("could not connect to context: %v", err)
			log.Fatal().Err(err).Msg("error connecting using nats context")
		}
	} else {
		nc, err = nats.Connect(opts.Server)
		if err != nil {
			log.Fatal().Err(err).Msg("error connecting to server")
		}
	}

	err = work(nc)
	if err != nil {
		log.Fatal().Err(err).Msg("error doing work")
	}
	fmt.Println("waiting")
	runtime.Goexit()
}

func addPortman(s string) error {
	parts := strings.Split(s, ":")
	if len(parts) < 2 {
		return fmt.Errorf("each mapping must contain 2 parts: %v", parts)
	}

	port, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("value '%s' is not parsable as port number: %W", parts[0], err)
	}

	opts.Portmap[port] = parts[1]
	return nil
}

func fileMappings(filename string) error {
	info, err := os.Stat(filename)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return errors.New("path is a directory")
	}

	fil, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fil.Close()

	s := bufio.NewScanner(fil)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		line := strings.Trim(s.Text(), " \t\r\n")
		if strings.HasPrefix(line, "#") {
			continue
		}
		if line == "" {
			continue
		}
		err = addPortman(line)
		if err != nil {
			return err
		}
	}
	return nil
}

func argMappings() error {
	if len(opts.Portmap) == 0 && flag.NArg() < 1 {
		return fmt.Errorf("You must provide at least one <port>:<subject> argument")
	}

	for _, pm := range flag.Args() {
		err := addPortman(pm)
		if err != nil {
			return err
		}
	}
	return nil
}
