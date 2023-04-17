package promnats

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

const (
	hdrAccept = "Accept"
	hdrPnID   = "Promnats-ID"
)

type options struct {
	RootSubject string
	Header      nats.Header
	Subjects    []string
	Subs        []*nats.Subscription
	Debug       bool
}

type Option func(*options) error

func testSafe(s string) error {
	var msgs []string
	if strings.ContainsAny(s, ".\r\n\t ") {
		msgs = append(msgs, "contains invalid chars")
	}

	if len(msgs) > 0 {
		return errors.New("invalid subject part: " + strings.Join(msgs, ","))
	}
	return nil
}

func WithSubj(parts ...string) Option {
	return func(o *options) error {
		if len(parts) > 0 {
			o.Subjects = []string{""}
		}
		for i, s := range parts {
			if err := testSafe(s); err != nil {
				return err
			}
			o.Subjects = append(o.Subjects, strings.Join(parts[:i], "."))
		}
		return nil
	}
}

func WithDebug() Option {
	return func(o *options) error {
		o.Debug = true
		return nil
	}
}

func execName() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	ex = filepath.Base(ex)
	ex = strings.TrimSuffix(ex, filepath.Ext(ex))
	return ex
}

func defaultSubjects() []string {
	hostname, _ := os.Hostname()
	return []string{
		"",
		execName(),
		hostname,
		strconv.Itoa(os.Getpid()),
	}
}

func RequestHandler(nc *nats.Conn, opts ...Option) error {
	//default
	cfg := options{
		RootSubject: "metrics",
		Header:      nats.Header{},
	}

	for _, o := range opts {
		err := o(&cfg)
		if err != nil {
			return err
		}
	}
	if len(cfg.Subjects) == 0 {
		cfg.Subjects = defaultSubjects()
	}

	cfg.Header.Add(hdrPnID, strings.Join(cfg.Subjects[1:], "."))

	reg := prometheus.ToTransactionalGatherer(prometheus.DefaultGatherer)

	handle := func(msg *nats.Msg) {
		err := handleMsg(msg, &cfg, reg)
		if err != nil {
			//TODO: notify shomehow
			if cfg.Debug {
				log.Printf("error handling message %v", err)
			}
		}
	}

	unsub := func() {
		for _, sub := range cfg.Subs {
			sub.Unsubscribe()
		}
	}
	if cfg.Debug {
		log.Printf("configured subjects %v", cfg.Subjects)
	}
	for _, subj := range cfg.Subjects {
		if subj != "" {
			subj = fmt.Sprintf("%s.%s", cfg.RootSubject, subj)
		} else {
			subj = cfg.RootSubject
		}
		sub, err := nc.Subscribe(subj, handle)
		if err != nil {
			unsub()
			return err
		}
		cfg.Subs = append(cfg.Subs, sub)
		if cfg.Debug {
			log.Printf("subscribing to %s", subj)
		}
	}

	return nil
}

func handleMsg(msg *nats.Msg, cfg *options, reg prometheus.TransactionalGatherer) error {
	mfs, done, err := reg.Gather()
	if err != nil {
		return err
	}
	defer done()

	contentType := negotiate(msg.Header)
	var buf bytes.Buffer
	enc := expfmt.NewEncoder(&buf, contentType)

	for _, mf := range mfs {
		err = enc.Encode(mf)
		if err != nil {
			return err
		}
	}
	if closer, ok := enc.(expfmt.Closer); ok {
		err = closer.Close()
		if err != nil {
			return err
		}
	}

	resp := nats.NewMsg(msg.Subject)
	resp.Header = cfg.Header
	resp.Header.Add("Content-Type", string(contentType))
	resp.Data = buf.Bytes()
	msg.RespondMsg(resp)

	return nil
}

func negotiate(h nats.Header) expfmt.Format {
	header := http.Header{}
	header.Add(hdrAccept, h.Get(hdrAccept))
	return expfmt.Negotiate(header)
}
