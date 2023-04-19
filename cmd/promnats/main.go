package main

import (
	"flag"
	"log"
	"time"

	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
)

type Opts struct {
	Timeout  time.Duration
	WithHTML bool
	Trace    bool
	Debug    bool
	Dest     string
	MaxAge   time.Duration
}

var (
	opts Opts = Opts{
		Timeout: 10 * time.Second,
	}
)

func main() {
	var contextFlag string
	var serverFlag string

	flag.StringVar(&contextFlag, "context", "", "context")
	flag.StringVar(&serverFlag, "server", "", "server like "+nats.DefaultURL)
	flag.BoolVar(&opts.Trace, "trace", false, "show trace")
	flag.BoolVar(&opts.Debug, "debug", false, "show debug")
	flag.BoolVar(&opts.WithHTML, "html", false, "output html")
	flag.StringVar(&opts.Dest, "dest", "./metrics", "folder to write output to")
	flag.DurationVar(&opts.MaxAge, "max-age", time.Hour, "how old info should we keep")

	flag.Parse()

	if opts.Debug {
		log.Printf("  opt.Timeout: %v\n", opts.Timeout)
		log.Printf("  Context: %s", contextFlag)
		log.Printf("  Args: %v", flag.Args())
	}

	if flag.NArg() != 1 {
		log.Fatalln("you must provide 1 argument as base subject")

	}
	var nc *nats.Conn
	var err error

	if serverFlag == "" {
		nc, err = natscontext.Connect(contextFlag)
		if err != nil {
			log.Fatalf("could not connect to context: %v", err)
		}
	} else {
		nc, err = nats.Connect(serverFlag)
		log.Fatalf("could not connect to server: %v", err)
	}
	work(nc)
}
