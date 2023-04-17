package promnatsplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type adsf struct {
	Trace bool
}

var (
	opts adsf = adsf{
		Trace: false,
	}
)

func doReqAsync(ctx context.Context, req any, subj string, waitFor int, nc *nats.Conn, cb func(*nats.Msg)) error {
	jreq := []byte("{}")
	var err error

	if req != nil {
		jreq, err = json.MarshalIndent(req, "", "  ")
		if err != nil {
			return err
		}
	}

	if opts.Trace {
		log.Printf(">>> %s: %s\n", subj, string(jreq))
	}

	var (
		mu  sync.Mutex
		ctr = 0
	)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var finisher *time.Timer
	if waitFor == 0 {
		finisher = time.NewTimer(300 * time.Millisecond)
		go func() {
			select {
			case <-finisher.C:
				cancel()
			case <-ctx.Done():
				return
			}
		}()
	}

	errs := make(chan error)
	sub, err := nc.Subscribe(nc.NewRespInbox(), func(m *nats.Msg) {
		mu.Lock()
		defer mu.Unlock()

		if opts.Trace {
			if m.Header != nil {
				log.Printf("<<< Header: %+v", m.Header)
			}
		}

		if finisher != nil {
			finisher.Reset(300 * time.Millisecond)
		}

		if m.Header.Get("Status") == "503" {
			errs <- nats.ErrNoResponders
			return
		}

		cb(m)
		ctr++

		if waitFor > 0 && ctr == waitFor {
			cancel()
		}
	})
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	if waitFor > 0 {
		sub.AutoUnsubscribe(waitFor)
	}

	msg := nats.NewMsg(subj)
	msg.Data = jreq
	msg.Reply = sub.Subject
	msg.Header.Add("Accept", "text/html")

	err = nc.PublishMsg(msg)
	if err != nil {
		return err
	}

	select {
	case err = <-errs:
		if err == nats.ErrNoResponders && strings.HasPrefix(subj, "$SYS") {
			return fmt.Errorf("server request failed, ensure the account used has system privileges and appropriate permissions")
		}

		return err
	case <-ctx.Done():
	}

	if opts.Trace {
		log.Printf("=== Received %d responses", ctr)
	}

	return nil
}

func doReq(ctx context.Context, req any, subj string, waitFor int, nc *nats.Conn) ([]*nats.Msg, error) {
	res := []*nats.Msg{}
	mu := sync.Mutex{}

	err := doReqAsync(ctx, req, subj, waitFor, nc, func(m *nats.Msg) {
		mu.Lock()
		res = append(res, m)
		mu.Unlock()
	})

	return res, err
}
