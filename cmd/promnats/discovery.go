package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
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

func discover(nc *nats.Conn, addr, host string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/discover", handleDiscovery(nc, 9000, host))

	server := &http.Server{
		Addr:    addr,
		Handler: WrapHandler(mux),
	}

	// run server in go func
	go func() {
		log.Info().Str("addr", server.Addr).Msg("discovery server started")
		err := server.ListenAndServe()
		if err != nil {
			// TODO: Should we try to restart or crash application?
			log.Error().Err(err).Any("server", server).Msg("discovery server died")
			panic(err)
		}
	}()

	return nil
}

type discovered struct {
	id    string
	parts []string
	port  int
}

func handleDiscovery(nc *nats.Conn, startport int, host string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		// read/update existing portmap
		err := fileMappings(opts.MappingFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		flipmap := map[string]int{}
		maxport := startport
		for port, pnid := range opts.Portmap {
			flipmap[pnid] = port
			if port > maxport {
				maxport = port
			}
		}
		maxport = maxport + 1

		msgs, err := doReq(r.Context(), nil, "metrics", 0, nc)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// groups key is part 0
		groups := map[string][]discovered{}
		nextport := maxport

		httpsd := []HttpEntry{}
		for _, m := range msgs {
			pnid := m.Header.Get(promnats.HeaderPnID)
			if pnid == "" {
				continue
			}
			port := 0
			for port == 0 {
				if p, ok := flipmap[pnid]; ok {
					//we have a port for that id, reuse
					port = p
				} else if _, ok := opts.Portmap[nextport]; !ok {
					//no such port is used... claim it
					port = nextport
					opts.Portmap[port] = pnid
					flipmap[pnid] = port
				} else {
					//try next port
					nextport = nextport + 1
				}
			}

			parts := strings.Split(pnid, ".")
			grpn := parts[0]
			d := discovered{id: pnid, parts: parts, port: port}
			if g, ok := groups[grpn]; ok {
				groups[grpn] = append(g, d)
			} else {
				groups[grpn] = []discovered{d}
			}
			log.Info().Str("pnid", pnid).Msg("something discovered")
		}

		for grpn, dg := range groups {
			entry := HttpEntry{
				Labels:  map[string]string{"__meta_prometheus_job": grpn, "subject_group": grpn},
				Targets: make([]string, 0),
			}
			for _, d := range dg {
				entry.Targets = append(entry.Targets, fmt.Sprintf("%s:%d", host, d.port))
			}
			httpsd = append(httpsd, entry)
		}
		w.Header().Add("Content-Type", "application/json")
		data, err := json.Marshal(&httpsd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = w.Write(data)
		if err != nil {
			log.Error().Err(err).Msg("error writing discovery response")
		}
		err = savePortmapSafe(opts.MappingFile)
		if err != nil {
			log.Error().Err(err).Msg("error saving portmap")
		} else {
			log.Info().Str("file", opts.MappingFile).Msg("portmap saved")
			err = work(nc)
			if err != nil {
				panic(err)
			}
		}

	}
}

func savePortmapSafe(filename string) error {
	newname := opts.MappingFile + ".new"
	defer func() {
		err := os.Remove(newname)
		if err != nil {
			log.Warn().Err(err).Msg("error removing safe file")
		}
	}()

	err := savePortmap(newname)
	if err != nil {
		return err
	}
	return copy(newname, opts.MappingFile)
}

func copy(from, to string) error {
	frominfo, err := os.Stat(from)
	if err != nil {
		return err
	}
	if !frominfo.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", from)
	}

	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(to)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func savePortmap(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	keys := make([]int, len(opts.Portmap))

	i := 0
	for k := range opts.Portmap {
		keys[i] = k
		i++
	}
	sort.Ints(keys)
	for _, k := range keys {
		pnid := opts.Portmap[k]
		_, err = w.WriteString(fmt.Sprintf("%d:%s\n", k, pnid))
		if err != nil {
			return err
		}
	}

	return w.Flush()
}
