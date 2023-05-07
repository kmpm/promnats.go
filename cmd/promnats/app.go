package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type application struct {
	servers  map[int]*http.Server
	server   *http.Server
	portmaps PortMaps
	mu       sync.Mutex
	closing  bool
	wg       sync.WaitGroup
	nc       *nats.Conn
}

func newApp() *application {
	return &application{
		servers:  make(map[int]*http.Server),
		portmaps: make(PortMaps),
	}
}

func (a *application) setServer(port int, srv *http.Server) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.servers[port]; ok {
		return fmt.Errorf("server with port %d already exists", port)
	}
	a.servers[port] = srv
	return nil
}

func (a *application) clearServer(port int) error {
	if s, ok := a.servers[port]; ok {
		err := s.Shutdown(context.Background())
		if err != nil {
			return err
		}
		delete(a.servers, port)
		delete(a.portmaps, port)
	}
	return nil
}

func (a *application) stop() error {

	a.closing = true
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	a.server.Shutdown(ctx)

	for _, s := range a.servers {
		err := s.Shutdown(ctx)
		if err != nil {
			log.Error().Err(err).Msg("error shutting down server")
		}
	}
	a.servers = nil
	a.wg.Wait()
	a.closing = false
	return nil

}

func (a *application) start(addr, host string, startport int) error {
	if a.server != nil {
		return fmt.Errorf("can not start a started application")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/discover", handleDiscovery(a.nc, startport, host, a.refresh))

	a.server = &http.Server{
		Addr:    addr,
		Handler: WrapHandler(mux),
	}
	a.wg.Add(1)
	// run server in go func
	go func() {
		defer a.wg.Done()
		log.Info().Str("addr", a.server.Addr).Msg("discovery server started")
		err := a.server.ListenAndServe()
		if err != nil {
			// TODO: Should we try to restart or crash application?
			if !a.closing {
				log.Panic().Err(err).Any("server", a.server.Addr).Msg("discovery server died")
			}
		}
	}()

	return nil
}
