
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
	go run ./cmd/promnats --context hermod-rfid metrics
	

.PHONY: caddywatch
caddyserver: $(CADDYBIN)
	$(CADDYBIN) run --watch --config testdata/Caddyfile


.PHONY: cleancaddy
cleancaddy: 
	del $(CADDYBIN)
	

$(CADDYBIN):
	xcaddy build --with github.com/kmpm/promnats.go/plugin/caddy

