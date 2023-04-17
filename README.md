# PromNATS - Prometheus reporting over NATS.
If you have loads of services interconnected using NATS and use it's benefits 
like automatic load balancing it can be annoying to use http for 
instrumentation with, in this case, prometheus.

This is a library that lets you use NATS requests instead of http requests
to return the prometheus data.


## Usage
```golang
func main() {
    nc, _ := nats.Connect(nats.DefaultURL)
    promnats.RequestHandler(nc)
}
```

```shell
nats req metrics ''

 nats --context nats_development req metrics ' '
19:28:21 Sending request on "metrics"
19:28:21 Received with rtt 1.5296ms
19:28:21 Promnats-ID: nats-demo-service.kmpm-ms-032d66.2264

# HELP go_gc_duration_seconds A summary of the pause duration of garbage collection cycles.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 0
go_gc_duration_seconds{quantile="0.25"} 0
go_gc_duration_seconds{quantile="0.5"} 0
go_gc_duration_seconds{quantile="0.75"} 0
go_gc_duration_seconds{quantile="1"} 0
go_gc_duration_seconds_sum 0
go_gc_duration_seconds_count 0
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 26
# HELP go_info Information about the Go environment.
# TYPE go_info gauge
go_info{version="go1.20.2"} 1
# HELP go_memstats_alloc_bytes Number of bytes allocated and still in use.
# TYPE go_memstats_alloc_bytes gauge
go_memstats_alloc_bytes 416072
# HELP go_memstats_alloc_bytes_total Total number of bytes allocated, even if freed.
# TYPE go_memstats_alloc_bytes_total counter
go_memstats_alloc_bytes_total 416072
...
```