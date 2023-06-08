package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/kmpm/promnats.go"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

// https://prometheus.io/docs/prometheus/latest/http_sd/
// https://prometheus.io/docs/guides/file-sd/#use-file-based-service-discovery-to-discover-scrape-targets
// https://stackoverflow.com/questions/44927130/different-prometheus-scrape-url-for-every-target

type HttpEntry struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

type discovered struct {
	id    string
	parts []string
	port  int
}

// handleDiscoryPaths create a http handler that returns a JSON for prometheus http service discovery
// that uses custome metrics_path instead of /metrics on different ports
func handleDiscoveryPaths(nc *nats.Conn, startport int, host string, refresh func(map[string]discovered) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// ask for data using a nats request
		log.Debug().Msg("disovering metrics for paths")
		var err error
		defer func() {
			log.Debug().Err(err).Msg("discovery paths done")
		}()
		var discoveries map[string]discovered
		discoveries, err = discoverPaths(r.Context(), nc, host, startport)
		httpsd := []HttpEntry{}
		for path, dg := range discoveries {
			grpn := dg.parts[0]
			entry := HttpEntry{
				Targets: []string{fmt.Sprintf("%s:%d", host, startport)},
				Labels: map[string]string{
					"__meta_prometheus_job": grpn,
					"subject_group":         grpn,
					"app":                   grpn,
					"task":                  dg.parts[2],
					"cluster":               dg.parts[1],
					"app_cluster":           strings.Join(dg.parts[:2], "."),
					"app_cluster_task":      strings.Join(dg.parts[:3], "."),
					"__metrics_path__":      "metrics/" + path,
				},
			}

			httpsd = append(httpsd, entry)
		}
		if r.URL.Query().Get("pretty") != "" {
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			err = enc.Encode(&httpsd)
		} else {
			var data []byte
			data, err = json.Marshal(&httpsd)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Add("Content-Type", "application/json")
			_, err = w.Write(data)
		}
		if err != nil {
			log.Error().Err(err).Msg("error writing discovery response")
			return
		}

		err = refresh(discoveries)
		if err != nil {
			log.Error().Err(err).Msg("error refreshing")
		}

	}
}

func discoverPaths(ctx context.Context, nc *nats.Conn, host string, port int) (discoveries map[string]discovered, err error) {
	var msgs []*nats.Msg
	discoveries = make(map[string]discovered)
	msgs, err = doReq(ctx, nil, "metrics", 0, nc)
	if err != nil {
		return
	}

	for _, m := range msgs {
		pnid := m.Header.Get(promnats.HeaderPnID)
		if pnid == "" {
			continue
		}
		parts := strings.Split(pnid, ".")
		d := discovered{id: pnid, parts: parts, port: port}
		path := strings.ToLower(strings.Join(parts, "/"))
		discoveries[path] = d
		log.Info().Str("pnid", pnid).Msg("something discovered")
	}
	return discoveries, nil
}
