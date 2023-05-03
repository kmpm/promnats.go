package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"sync"

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
	Context       string
	Server        string
	Nkey          string
	Timeout       time.Duration
	Verbosity     string
	Portmap       map[int]string
	MappingFile   string
	PrettyLog     bool
	Servers       []*http.Server
	Closing       bool
	serversWg     sync.WaitGroup
	DiscoveryHost string
}

var opts Opts
var mu sync.Mutex

func init() {
	// initialize opts
	opts = Opts{
		Portmap: make(map[int]string),
		Servers: make([]*http.Server, 0),
	}
	mu.Lock()
	defer mu.Unlock()
	// add some flags for connection
	flag.StringVar(&opts.Context, "context", "", "context")
	flag.StringVar(&opts.Server, "server", "", "server like "+nats.DefaultURL)
	flag.StringVar(&opts.Nkey, "nkey", "", "pathto nkey file")

	// flags for other config
	flag.StringVar(&opts.Verbosity, "verbosity", "info", "debug|info|warn|error")
	flag.DurationVar(&opts.Timeout, "timeout", time.Second*2, "time waiting for replies")
	flag.StringVar(&opts.MappingFile, "mapping", "", "path to file with <port>:<subject> mappings instead of args")
	flag.BoolVar(&opts.PrettyLog, "pretty", false, "pretty logging")

	flag.StringVar(&opts.DiscoveryHost, "discovery", "", "ip / hostname to show in http_sd")
}

func main() {
	flag.Parse()
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

	var err error
	// load mappings from file
	if opts.MappingFile != "" {
		mu.Lock()
		err = fileMappings(opts.MappingFile)
		if err != nil {
			log.Fatal().Err(err).Str("mapping", opts.MappingFile).Msg("error reading mappings")
		}
		mu.Unlock()
	}

	// get other mappings from args
	mu.Lock()
	err = argMappings()
	if err != nil {
		log.Fatal().Err(err).Msg("error parsing arguments")
	}
	mu.Unlock()

	// open connection by context or server
	if opts.Server != "" && opts.Context != "" {
		log.Fatal().Msg("you must not use both context and server")
	}
	var nc *nats.Conn
	if opts.Server == "" {
		nc, err = natscontext.Connect(opts.Context)
		if err != nil {
			log.Fatal().Err(err).Msg("error connecting using nats context")
		}
	} else {
		nc, err = nats.Connect(opts.Server)
		if err != nil {
			log.Fatal().Err(err).Msg("error connecting to server")
		}
	}

	err = discover(nc, ":8083", opts.DiscoveryHost)
	if err != nil {
		log.Fatal().Err(err).Msg("error discovering")
	}

	if len(opts.Portmap) > 0 {
		// do aktual work
		err = work(nc)
		if err != nil {
			log.Fatal().Err(err).Msg("error doing work")
		}
	}

	// wait for things to close
	fmt.Println("waiting")
	runtime.Goexit()
}

// addPortman parses a string and adds to opts.Portmap if valid.
func addPortmap(s string) error {
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

// fileMappings reads filename, line by line and adds them to portmap if valid
func fileMappings(filename string) error {

	// check filename if path exists
	info, err := os.Stat(filename)
	if err != nil {
		return err
	}
	// cant be a dir
	if info.IsDir() {
		return errors.New("path is a directory")
	}
	// open the file
	fil, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer fil.Close()

	// read line by line
	s := bufio.NewScanner(fil)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		// trim any bad characters
		line := strings.Trim(s.Text(), " \t\r\n")
		// skip # comment lines
		if strings.HasPrefix(line, "#") {
			continue
		}
		// skip empty lines
		if line == "" {
			continue
		}
		// parse and possibly add
		err = addPortmap(line)
		if err != nil {
			return err
		}
	}
	return nil
}

// argMappings adds arguments to anything that is allready in portmap
// if portmap is empty there must be att least 1 argument with portmap config
func argMappings() error {
	// if len(opts.Portmap) == 0 && flag.NArg() < 1 {
	// 	return fmt.Errorf("You must provide at least one <port>:<subject> argument")
	// }

	for _, pm := range flag.Args() {
		err := addPortmap(pm)
		if err != nil {
			return err
		}
	}
	return nil
}
