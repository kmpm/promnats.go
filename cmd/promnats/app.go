package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type application struct {
	servers map[int]*http.Server
	server  *http.Server

	discoveries map[string]discovered
	mu          sync.Mutex
	closing     bool
	wg          sync.WaitGroup
	nc          *nats.Conn
}

func newApp() *application {
	return &application{
		servers:     make(map[int]*http.Server),
		discoveries: map[string]discovered{},
	}
}

func (a *application) stop() error {

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

// shiftPath splits the given path into the first segment (head) and
// the rest (tail). For example, "/foo/bar/baz" gives "foo", "/bar/baz".
func shiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}

func (a *application) start(addr, host string, startport int) error {
	if a.server != nil {
		return fmt.Errorf("can not start a started application")
	}
	mux := http.NewServeMux()
	// mux.HandleFunc("/discover", handleDiscovery(a.nc, startport, host, a.refresh))
	handleDiscovery := handleDiscoveryPaths(a.nc, startport, host, a.refreshPaths)
	handlePath := a.makePathHandler()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var head string
		head, r.URL.Path = shiftPath(r.URL.Path)
		// log.Debug().Str("head", head).Str("path", r.URL.Path).Msg("shifted path")
		switch head {
		case "metrics":
			if len(a.discoveries) == 0 {
				go func() {
					paths, err := discoverPaths(context.Background(), a.nc, startport)
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
	})

	a.server = &http.Server{
		Addr:    addr,
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
