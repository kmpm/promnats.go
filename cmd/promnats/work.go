package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// work sets up the http handlers for each entry in portmap
func (a *application) refresh(pm PortMaps) error {
	if len(pm) > 0 {
		a.portmaps = pm
	}
	running := make([]int, 0)
	for port, subj := range a.portmaps {
		if _, ok := a.servers[port]; ok {
			running = append(running, port)
			continue
		}

		// create a multiplexer with metrics handler
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", a.makeHandler(port))
		// create a http.Server with given port
		s := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: WrapHandler(mux),
		}
		err := a.setServer(port, s)
		check(err)
		running = append(running, port)

		// run server in go func
		a.wg.Add(1)
		go func(subj string) {
			defer a.wg.Done()
			log.Info().Str("addr", s.Addr).Str("subj", subj).Msg("value server listening")
			err := s.ListenAndServe()
			if err != nil {
				if !a.closing {
					log.Warn().Err(err).Str("server", s.Addr).Str("subj", subj).Msg("value server died")
				}

			}
		}(subj)
	}

	var found bool
	var toDelete []int
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for serverport, s := range a.servers {
		found = false
		for _, runningport := range running {
			if serverport == runningport {
				found = true
				break
			}
		}
		if !found {
			s.Shutdown(ctx)
			toDelete = append(toDelete, serverport)
		}
	}
	for _, port := range toDelete {
		delete(a.servers, port)
		delete(a.portmaps, port)
	}
	return nil
}

func (a *application) makeHandler(port int) func(http.ResponseWriter, *http.Request) {
	// return a http handler
	return func(w http.ResponseWriter, r *http.Request) {
		// save start-time to be able to calculate response time later
		start := time.Now()
		subj, ok := a.portmaps[port]
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
		}
		// send nats request with context from http.Request
		// wait for first answer
		msgs, err := doReq(r.Context(), nil, "metrics."+subj, 1, a.nc)
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
