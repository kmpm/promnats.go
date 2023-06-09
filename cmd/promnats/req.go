package main

import (
	"context"
	"encoding/json"
	"fmt"

	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
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
	log.Debug().Str("subj", subj).Msg("inbound")

	var (
		mu  sync.Mutex
		ctr = 0
	)

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
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

		log.Debug().Any("headers", m.Header).Msg("inbound")

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

	log.Debug().Int("responses", ctr).Msg("receive complete")

	return nil
}

// doReq sends request to subject and return any replies
// stops at opts.Timeout or waitFor number of replies if > 0
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
