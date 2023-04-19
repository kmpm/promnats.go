
GOOS=$(shell go env GOOS)



ifeq ($(GOOS),windows) 
	BINEXT = .exe
else
	BINEXT =
endif


CADDYBIN=caddy$(BINEXT)



test:
	go test ./...

.PHONY: collect
collect:
	go run ./cmd/promnats --max-age 5m --context hermod-rfid metrics 
	
