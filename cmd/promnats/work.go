package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// work sets up the http handlers for each entry in portmap
func work(nc *nats.Conn) error {
	// loop every entry in opts.Portmap
	for port, subj := range opts.Portmap {
		// create a multiplexer with metrics handler
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", makeHandler(subj, nc))

		// create a http.Server with given port
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		}

		// run server in go func
		go func(subj string) {
			log.Info().Str("addr", server.Addr).Str("subj", subj).Msg("listening")
			err := server.ListenAndServe()
			if err != nil {
				// TODO: Should we try to restart or crash application?
				log.Error().Err(err).Any("server", server).Msg("server died")
				panic(err)
			}
		}(subj)
	}
	return nil
}

func makeHandler(subj string, nc *nats.Conn) func(http.ResponseWriter, *http.Request) {
	// return a http handler
	return func(w http.ResponseWriter, r *http.Request) {
		// save start-time to be able to calculate response time later
		start := time.Now()
		// send nats request with context from http.Request
		// wait for first answer
		msgs, err := doReq(r.Context(), nil, subj, 1, nc)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Error().Err(err).Str("subj", subj).Msg("doReq error")
			return
		}

		// should have at least one message
		if len(msgs) < 1 {
			http.Error(w, fmt.Sprintf("%s not found", subj), http.StatusNotFound)
			log.Warn().Str("subj", subj).Msg("not found")
			return
		}

		// get the first message
		msg := msgs[0]

		// add headers if we have them
		w.Header().Add("X-Promnats-ID", msg.Header.Get("Promnats-ID"))
		if ct := msg.Header.Get("Content-Type"); ct != "" {
			w.Header().Add("Content-Type", ct)
		}
		// respond with data
		size, err := w.Write(msg.Data)
		if err != nil {
			log.Warn().Err(err).Str("subj", subj).Dur("response_time", time.Since(start)).Msg("error responding")
		} else {
			log.Debug().Str("subj", subj).Int("size", size).Err(err).Dur("response_time", time.Since(start)).Msg("responding")
		}
	}
}
