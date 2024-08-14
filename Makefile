
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
RUNARGS?=-verbosity debug
NAME=promnats
CNREPO?=your.docker.repo
CNNAME=$(CNREPO)/$(NAME)

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
	PWD = $(subst \,/,$(subst :,:,$(shell cd)))
else
	FixPath = $1
	MKDIR = mkdir -p
	RM = rm
	BINEXT = 
	CP = cp
endif

# version stuff
VERSION?=$(shell git describe --tags --always --long --dirty)
word-dot = $(word $2,$(subst ., ,$1))
word-dash = $(word $2,$(subst -, ,$1))
MAJOR=$(subst v,,$(call word-dot,$(VERSION),1))
MINOR=$(call word-dot,$(VERSION),2)
REVISION=$(call word-dash,$(call word-dot,$(VERSION),3),1)
PATCH=$(call word-dash,$(VERSION),2)

# during build
OUT_DIR=./out
OUT_FILE=$(call FixPath,$(OUT_DIR)/promnats-$(GOOS)-$(GOARCH)$(BINEXT))
OUTVERBOSE_FILE=$(call FixPath,$(OUT_DIR)/promnats-$(GOOS)-$(GOARCH)-$(VERSION)$(BINEXT))
GOFLAGS=-ldflags "-X 'main.appVersion=$(VERSION)'"

help:
	@echo VERSION = $(VERSION)


.PHONY: build
build: $(OUT_DIR)
	go build $(GOFLAGS) -o $(OUTVERBOSE_FILE) $(call FixPath,./cmd/promnats)
	$(CP) $(OUTVERBOSE_FILE) $(OUT_FILE)


$(OUT_DIR):
	$(MKDIR) $(call FixPath,$(OUT_DIR))


test:
	go test ./...

.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

.PHONY: audit
audit:
	@echo "running checks"
	go mod verify
	go vet ./...
	go list -m all
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...


.PHONY: run
run:
	go run $(GOFLAGS) ./cmd/promnats $(RUNARGS)
	

.PHONY: image
image: tidy
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg APPNAME=$(NAME) \
		-t $(NAME):latest \
		-f Dockerfile .
		
.PHONY: tags
tags: no-dirty
	docker tag $(NAME):latest $(CNNAME):$(VERSION)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR).$(MINOR)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR).$(MINOR).$(REVISION)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR).$(MINOR).$(REVISION)-$(PATCH)

.PHONY: push
push: tidy audit no-dirty
	docker push -a $(CNNAME) 

.PHONY: edge
edge: tidy image
	docker tag $(NAME):latest $(CNNAME):edge
	
.PHONY: testserver
testserver:
	docker run -it --rm \
		-p 9090:9090 \
		-v "$(call FixPath,$(PWD)/contrib/prometheus):/etc/prometheus/contrib" \
		prom/prometheus \
		--config.file=/etc/prometheus/contrib/prometheus.yml

.PHONY: no-dirty
no-dirty:
	git diff --exit-code
