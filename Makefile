
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
RUNARGS?=-verbosity debug -mapping .\mappings-secret.txt -context hermod-rfid

ifeq ($(GOOS),windows) 
	BINEXT = .exe
else
	BINEXT =
endif

# try to be os agnostic
ifeq ($(OS),Windows_NT)
	FixPath = $(subst /,\,$1)
	RM = del
	MKDIR = mkdir
	CP = copy
else
	FixPath = $1
	MKDIR = mkdir -p
	RM = rm
	BINEXT = 
	CP = cp
endif



VERSION?=$(shell git describe --tags --always --long --dirty)
OUT_DIR=./out
OUT_FILE=$(call FixPath,$(OUT_DIR)/promnats-$(GOOS)-$(GOARCH)$(BINEXT))
OUTVERBOSE_FILE=$(call FixPath,$(OUT_DIR)/promnats-$(GOOS)-$(GOARCH)-$(VERSION)$(BINEXT))

GOFLAGS=-ldflags "-X 'main.appVersion=$(VERSION)'"

.PHONY: build
build: $(OUT_DIR)
	go build $(GOFLAGS) -o $(OUTVERBOSE_FILE) $(call FixPath,./cmd/promnats)
	$(CP) $(OUTVERBOSE_FILE) $(OUT_FILE)

$(OUT_DIR):
	$(MKDIR) $(call FixPath,$(OUT_DIR))


test:
	go test ./...


.PHONY: run
run:
	go run $(GOFLAGS) ./cmd/promnats $(RUNARGS)
	
