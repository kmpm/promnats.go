package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/nats-io/nats.go"
)

func work(nc *nats.Conn) error {
	// servers := []*http.Server{}

	for port, subj := range opts.Portmap {
		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: makeHandler(subj, nc),
		}
		// servers = append(servers, server)
		go func(subj string) {
			log.Printf("listening to %s for %s", server.Addr, subj)
			err := server.ListenAndServe()
			if err != nil {
				log.Printf("server died %v: %v", server, err)
			}
		}(subj)
	}
	return nil
}

func makeHandler(subj string, nc *nats.Conn) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msgs, err := doReq(context.TODO(), nil, subj, 1, nc)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if len(msgs) < 1 {
			http.Error(w, fmt.Sprintf("%s not found", subj), http.StatusNotFound)
			return
		}

		// w.Header().Add("Content-Type", "text/plain")
		w.Write(msgs[0].Data)
	})
}
