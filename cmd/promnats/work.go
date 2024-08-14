package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

func (a *application) refreshPaths(discoveries map[string]discovered) error {
	a.mu.Lock()
	defer func() {
		slog.Debug("refreshPaths done")
		a.mu.Unlock()
	}()
	slog.Debug("refreshPaths")
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
			slog.Warn("not found", "path", r.URL.Path, "key", key, "discoveries", a.discoveries)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		subj := disc.id
		// send nats request with context from http.Request
		// wait for first answer
		msgs, err := doReq(r.Context(), nil, "metrics."+subj, 1, a.nc)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			slog.Error("doReq error", "error", err, "subject", subj)
			return
		}

		// should have at least one message
		if len(msgs) < 1 {
			http.Error(w, fmt.Sprintf("%s not found", subj), http.StatusNotFound)
			slog.Warn("not found", "subject", subj)
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
			slog.Warn("error responding", "error", err, "subject", subj, "response_time", time.Since(start))
		} else {
			slog.Debug("responding", "subject", subj, "size", size, "response_time", time.Since(start), "error", err)
		}
	}
}
