package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kmpm/promnats.go/internal/impl/req"
	"github.com/nats-io/nats.go"
)

type App struct {
	servers   map[int]*http.Server
	server    *http.Server
	timeout   time.Duration
	addr      string
	host      string
	startport int

	discoveries map[string]discovered
	mu          sync.Mutex
	closing     bool
	wg          sync.WaitGroup
	nc          *nats.Conn
}

func newApp(timeout time.Duration, host, addr string, startport int) *App {
	return &App{
		servers:     make(map[int]*http.Server),
		discoveries: map[string]discovered{},
		timeout:     timeout,
		addr:        addr,
		host:        host,
		startport:   startport,
	}
}

func (a *App) stop() error {

	a.closing = true
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a.server.Shutdown(ctx)

	for _, s := range a.servers {
		err := s.Shutdown(ctx)
		if err != nil {
			slog.Error("error shutting down server", "error", err)
		}
	}
	a.servers = nil
	a.wg.Wait()
	a.closing = false
	return nil

}

func (a *App) start() error {
	if a.server != nil {
		return fmt.Errorf("can not start a started application")
	}
	mux := http.NewServeMux()
	// mux.HandleFunc("/discover", handleDiscovery(a.nc, startport, host, a.refresh))

	mux.HandleFunc("/", a.makeRootHandler())

	a.server = &http.Server{
		Addr:    a.addr,
		Handler: WrapHandler(mux),
	}
	a.wg.Add(1)
	// run server in go func
	go func() {
		defer a.wg.Done()
		slog.Info("discovery server started", "addr", a.server.Addr)
		err := a.server.ListenAndServe()
		if err != nil {
			// panic if not closing
			if !a.closing {
				panic(fmt.Errorf("discovery server died: %w", err))
			}
		}
	}()

	return nil
}

func (a *App) refreshPaths(discoveries map[string]discovered) error {
	a.mu.Lock()
	defer func() {
		slog.Debug("refreshPaths done")
		a.mu.Unlock()
	}()
	slog.Debug("refreshPaths")
	a.discoveries = discoveries
	return nil
}

func (a *App) makePathHandler() func(http.ResponseWriter, *http.Request) {
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
		msgs, err := req.Do(r.Context(), nil, "metrics."+subj, 1, a.timeout, a.nc)
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

func (a *App) makeRootHandler() func(http.ResponseWriter, *http.Request) {
	handleDiscovery := handleDiscoveryPaths(a.nc, a.startport, a.host, a.timeout, a.refreshPaths)
	handlePath := a.makePathHandler()

	return func(w http.ResponseWriter, r *http.Request) {
		var head string
		head, r.URL.Path = shiftPath(r.URL.Path)
		// log.Debug().Str("head", head).Str("path", r.URL.Path).Msg("shifted path")
		switch head {
		case "metrics":
			if a.discoveries == nil || len(a.discoveries) == 0 {
				go func() {
					paths, err := discoverPaths(context.Background(), a.nc, a.host, a.startport, a.timeout)
					if err != nil {
						slog.Error("error discovering paths", "error", err)
						return
					}
					err = a.refreshPaths(paths)
					if err != nil {
						slog.Error("error refreshing paths", "error", err)
						return
					}
				}()
			}
			handlePath(w, r)
			return
		case "discover":
			handleDiscovery(w, r)
			return
		}
		http.NotFound(w, r)
	}
}
