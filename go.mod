module github.com/kmpm/promnats.go

go 1.20

require (
	github.com/kmpm/flagenvfile.go v0.0.0-00010101000000-000000000000
	github.com/nats-io/jsm.go v0.0.35
	github.com/nats-io/nats.go v1.25.0
	github.com/prometheus/client_golang v1.15.0
	github.com/prometheus/common v0.42.0
	github.com/rs/zerolog v1.29.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.18 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/nats-io/nats-server/v2 v2.9.16 // indirect
	github.com/nats-io/nkeys v0.4.4 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	golang.org/x/crypto v0.8.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)

replace github.com/kmpm/flagenvfile.go => ../flagenvfile.go
