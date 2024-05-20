
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
RUNARGS?=-verbosity debug
RUNDIR?=${CURDIR}/cmd/promnats
NAME=promnats
CNREPO?=your.docker.repo
CNNAME=$(CNREPO)/$(NAME)


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

CMDS=cmd/promnats cmd/promnatsdp


# during build
OUT_DIR=./out/$(GOOS)-$(GOARCH)
GOFLAGS=-ldflags "-X 'main.appVersion=$(VERSION)'"

help:
	@echo VERSION = $(VERSION)


.PHONY: build
build: $(CMDS)
	@echo "Build complete in $(OUT_DIR)"

$(CMDS): $(OUT_DIR)
	cd $(OUT_DIR) ; go build $(GOFLAGS) $(call FixPath,$(CURDIR)/$@)


$(OUT_DIR):
	$(MKDIR) $(call FixPath,$(OUT_DIR))


test:
	go test ./...


.PHONY: tidy
tidy:
	go mod tidy


.PHONY: run
run:
	go run $(GOFLAGS) $(RUNDIR) $(RUNARGS)
	

.PHONY: image
image:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg APPNAME=$(NAME) \
		-t $(NAME):latest \
		-f Dockerfile .
		
.PHONY: tags
tags:
	docker tag $(NAME):latest $(CNNAME):$(VERSION)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR).$(MINOR)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR).$(MINOR).$(REVISION)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR).$(MINOR).$(REVISION)-$(PATCH)

.PHONY: push
push:
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