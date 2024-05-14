package cli

import (
	"flag"
	"time"

	"github.com/kmpm/flagenvfile.go"
	"github.com/nats-io/nats.go"
)

type Options struct {
	Context string
	Server  string
	Nkey    string
	Timeout time.Duration

	Address string
	Host    string
	// Version bool

	Verbosity string
	// PrettyLog bool
}

func newOptionsFromFEF() (*Options, error) {
	flag.String("context", "", "context")
	flag.String("server", "", "server like "+nats.DefaultURL)
	flag.String("nkey", "", "pathto nkey file")

	flag.Duration("timeout", time.Second*2, "time waiting for replies")
	flag.String("address", ":8083", "address to listen on")
	flag.String("host", "", "host to use for http_sd. defaults to local IP if only 1")
	// flag.Bool("version", false, "show version and exit")
	// flags for other config
	flag.String("verbosity", "info", "debug|info|warn|error")
	flag.Bool("pretty", false, "pretty logging")

	opts := &Options{}
	flag.Parse()
	flagenvfile.BindFlagset(flag.CommandLine)
	flagenvfile.SetEnvPrefix("PN")

	opts.Verbosity = flagenvfile.GetString("verbosity")
	// opts.PrettyLog = flagenvfile.GetBool("pretty")
	// opts.Version = flagenvfile.GetBool("version")
	opts.Address = flagenvfile.GetString("address")
	opts.Context = flagenvfile.GetString("context")
	opts.Server = flagenvfile.GetString("server")
	opts.Nkey = flagenvfile.GetString("nkey")
	opts.Host = flagenvfile.GetString("host")
	opts.Timeout = flagenvfile.GetDuration("timeout")
	return opts, nil
}
