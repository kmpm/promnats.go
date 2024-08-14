package promnats

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

const (
	hdrAccept  = "Accept"
	HeaderPnID = "Promnats-ID"
)

type options struct {
	RootSubject string
	Header      nats.Header
	Subjects    []string
	Subs        []*nats.Subscription
	Debug       bool
	ID          string
}

type Option func(*options) error

func testSafe(s string) error {
	var msgs []string
	if strings.ContainsAny(s, ".\r\n\t ") {
		msgs = append(msgs, "contains invalid chars")
	}
	if len(s) < 1 {
		msgs = append(msgs, "zero length")
	}
	if len(msgs) > 0 {
		return fmt.Errorf("invalid subject part '%s': %s", s, strings.Join(msgs, ","))
	}
	return nil
}

func HostPart() string {
	s, _ := os.Hostname()
	s = strings.ReplaceAll(s, ".", "_")
	return strings.ToLower(s)
}

func ExecPart() string {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	ex = filepath.Base(ex)
	ex = strings.TrimSuffix(ex, filepath.Ext(ex))
	return strings.ToLower(strings.ReplaceAll(ex, ".", "_"))
}

func PidPart() string {
	return strconv.Itoa(os.Getpid())
}

// WithParts will generate one subscription for each part
func WithParts(parts ...string) Option {
	return func(o *options) error {
		if l := len(parts); l < 1 {
			return errors.New("must be at least 1 part")
		}
		if len(parts) > 0 {
			o.Subjects = []string{""}
		}
		for i, s := range parts {
			if err := testSafe(s); err != nil {
				return err
			}
			o.Subjects = append(o.Subjects, strings.ToLower(strings.Join(parts[:i+1], ".")))
		}
		return nil
	}
}

// WithID will split input at . and create a subscription for each part
func WithID(id string) Option {
	return WithParts(strings.Split(id, ".")...)
}

func WithDebug() Option {
	return func(o *options) error {
		o.Debug = true
		return nil
	}
}
func DefaultSubject() string {
	return ExecPart() + "." + HostPart() + "." + PidPart()
}

func defaultSubjects() []string {
	return []string{
		"",
		ExecPart(),
		ExecPart() + "." + HostPart(),
		ExecPart() + "." + HostPart() + "." + PidPart(),
	}
}

func genID(s []string) string {
	return strings.ToLower(s[len(s)-1])
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
	cfg.ID = genID(cfg.Subjects)
	cfg.Header.Add(HeaderPnID, cfg.ID)

	reg := prometheus.ToTransactionalGatherer(prometheus.DefaultGatherer)

	handle := func(msg *nats.Msg) {
		err := handleMsg(msg, &cfg, reg)
		if err != nil {
			//TODO: notify shomehow
			if cfg.Debug {
				slog.Debug("error handling message", "err", err)
			}
		}
	}

	unsub := func() {
		for _, sub := range cfg.Subs {
			sub.Unsubscribe()
		}
	}
	if cfg.Debug {
		slog.Debug("configured subjects", "subjects", cfg.Subjects)
	}
	for _, subj := range cfg.Subjects {
		if subj != "" {
			subj = fmt.Sprintf("%s.%s", cfg.RootSubject, strings.ToLower(subj))
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
			slog.Debug("subscribing to", "subject", subj)
		}
	}

	return nil
}

func handleMsg(msg *nats.Msg, cfg *options, reg prometheus.TransactionalGatherer) error {
	start := time.Now()
	if cfg.Debug {
		defer func() {
			slog.Debug("promnats response time", "time", time.Since(start))
		}()
	}
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

	resp.Header.Set("Content-Type", string(contentType))
	// if cfg.Debug {
	// 	log.Printf("response: %v", resp)
	// }
	resp.Data = buf.Bytes()

	err = msg.RespondMsg(resp)
	if err != nil {
		//log error
		slog.Error("error sending reply", "err", err)
	}

	return nil
}

func negotiate(h nats.Header) expfmt.Format {
	header := http.Header{}
	header.Add(hdrAccept, h.Get(hdrAccept))
	return expfmt.Negotiate(header)
}
