package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

func work(nc *nats.Conn) error {
	// servers := []*http.Server{}

	for port, subj := range opts.Portmap {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", makeHandler(subj, nc))

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		}
		// servers = append(servers, server)
		go func(subj string) {
			log.Info().Str("addr", server.Addr).Str("subj", subj).Msg("listening")
			err := server.ListenAndServe()
			if err != nil {
				log.Error().Err(err).Any("server", server).Msg("server died")
			}
		}(subj)
	}
	return nil
}

func makeHandler(subj string, nc *nats.Conn) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		msgs, err := doReq(context.TODO(), nil, subj, 1, nc)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Error().Err(err).Str("subj", subj).Msg("doReq error")
			return
		}

		if len(msgs) < 1 {
			http.Error(w, fmt.Sprintf("%s not found", subj), http.StatusNotFound)
			log.Warn().Str("subj", subj).Msg("not found")
			return
		}
		msg := msgs[0]
		w.Header().Add("X-Promnats-ID", msg.Header.Get("Promnats-ID"))
		if ct := msg.Header.Get("Content-Type"); ct != "" {
			w.Header().Add("Content-Type", ct)
		}
		size, err := w.Write(msg.Data)
		log.Debug().Str("subj", subj).Int("size", size).Err(err).Dur("response_time", time.Since(start)).Msg("responding")
		if err != nil {
			log.Warn().Err(err).Str("subj", subj).Dur("response_time", time.Since(start)).Msg("error responding")
		}
	}
}
