package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metHTTPRequestCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "promnats_http_requests_total",
		Help: "Total number of requests over https",
	})

	metSubGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "promnats_subs_total",
		Help: "Total number nats subscriptions",
	})

	metPubCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name: "promnats_pubs_total",
		Help: "Total number of published messages on nats",
	})

	metDiscoveredPaths = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "promnats_discovered_total",
		Help: "Total number discovered hosts",
	})

	metPathRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "promnats_paths_total",
			Help: "How many path requests processed, partitioned by subject",
		},
		[]string{"subject"},
	)

	metPathFails = promauto.NewCounter(prometheus.CounterOpts{
		Name: "promnats_path_fails",
		Help: "Total number of path requests failed",
	})
)
