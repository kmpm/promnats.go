package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

func (a *application) refreshPaths(discoveries map[string]discovered) error {
	a.mu.Lock()
	defer func() {
		log.Debug().Msg("refreshPaths done")
		a.mu.Unlock()
	}()
	log.Debug().Msg("refreshPaths")
	a.discoveries = discoveries
	return nil
}

func (a *application) makePathHandler() func(http.ResponseWriter, *http.Request) {
	// return a http handler
	return func(w http.ResponseWriter, r *http.Request) {
		// save start-time to be able to calculate response time later
		start := time.Now()

		key := strings.TrimPrefix(r.URL.Path, "/")

		disc, ok := a.discoveries[key]
		if !ok {
			log.Warn().Str("path", r.URL.Path).Str("key", key).Any("dsc", a.discoveries).Msg("not found")
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		subj := disc.id
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
