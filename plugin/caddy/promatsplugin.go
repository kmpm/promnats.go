package promnatscaddy

import (
	"fmt"
	"net/http"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/nats-io/jsm.go/natscontext"
	"github.com/nats-io/nats.go"
)

func init() {
	caddy.RegisterModule(PromNats{})
}

// CaddyModule returns the Caddy module information.
func (PromNats) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.promnats",
		New: func() caddy.Module { return newModule() },
	}
}

func newModule() *PromNats {
	pn := &PromNats{
		Interval: time.Minute * 5,
	}
	return pn
}

func (m *PromNats) Provision(ctx caddy.Context) error {
	m.logger = ctx.Logger()
	var err error
	var nc *nats.Conn
	if m.ServerURL == "" {
		nc, err = natscontext.Connect(m.ContextName)
	} else {
		nc, err = nats.Connect(m.ServerURL)
	}
	if err != nil {
		return err
	}

	m.nc = nc
	m.refresh()
	return nil
}

// Validate implements caddy.Validator.
func (m *PromNats) Validate() error {
	if m.ContextName == "" && m.ServerURL == "" {
		return fmt.Errorf("no context or server")
	}
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m *PromNats) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	// m.w.Write([]byte(r.RemoteAddr))

	return next.ServeHTTP(w, r)
}

// Interface guards
var (
	_ caddy.Provisioner           = (*PromNats)(nil)
	_ caddy.Validator             = (*PromNats)(nil)
	_ caddyhttp.MiddlewareHandler = (*PromNats)(nil)
	_ caddyfile.Unmarshaler       = (*PromNats)(nil)
)
