
RUNARGS?=-verbosity debug
CNREPO?=your.docker.repo
NAME=promnats
CNNAME:=${CNREPO}/${NAME}
GOOS:=$(shell go env GOOS)
GOARCH:=$(shell go env GOARCH)
GOEXE:=$(shell go env GOEXE)


# try to be os agnostic
ifeq ($(OS),Windows_NT)
	FixPath = $(subst /,\,$1)
	RM = del /Q
	MKDIR = mkdir
	CP = copy
	PWD = $(subst \,/,$(subst :,:,$(shell cd)))
else
	FixPath = $1
	MKDIR = mkdir -p
	RM = rm -Rf	
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
OUT_FILE=$(call FixPath,$(OUT_DIR)/promnats-$(GOOS)-$(GOARCH)$(GOEXE))
GOFLAGS=-ldflags "-X 'main.appVersion=$(VERSION)'"

help: info
	@echo "make build - build the promnats binary"
	@echo "make test - run tests"
	@echo "make tidy - run go fmt and go mod tidy"
	@echo "make audit - run go mod verify, go vet, staticcheck, and go vuln check"

info:
	@echo "GOOS=$(GOOS)"
	@echo "GOARCH=$(GOARCH)"
	@echo "GOEXE=$(GOEXE)"
	@echo "VERSION=$(VERSION)"
	@echo "NAME=$(NAME)"
	@echo "CNREPO=${CNREPO}"
	@echo "CNNAME=${CNNAME}"
	@echo tag "$(CNNAME):$(MAJOR).$(MINOR).$(REVISION)-$(PATCH)"


.PHONY: build
build: $(OUT_DIR)
	go build $(GOFLAGS) -o $(OUT_FILE) $(call FixPath,./cmd/promnats)


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
image:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg APPNAME=$(NAME) \
		-t $(NAME):latest \
		-t "$(CNNAME):$(VERSION)" \
		-t "$(CNNAME):$(MAJOR)" \
		-t "$(CNNAME):$(MAJOR).$(MINOR)" \
		-t "$(CNNAME):$(MAJOR).$(MINOR).$(REVISION)" \
		-t "$(CNNAME):$(MAJOR).$(MINOR).$(REVISION)-$(PATCH)" \
		-f Dockerfile .
		
.PHONY: tags
tags: no-dirty
	docker tag $(NAME):latest $(CNNAME):$(VERSION)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR).$(MINOR)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR).$(MINOR).$(REVISION)
	docker tag $(NAME):latest $(CNNAME):$(MAJOR).$(MINOR).$(REVISION)-$(PATCH)

.PHONY: push
push: tidy audit no-dirty tags edge
	docker push -a $(CNNAME) 


.PHONY: edge
edge: tidy image
	docker tag $(NAME):latest $(CNNAME):edge


.PHONY: no-dirty
no-dirty:
	git diff --exit-code


.PHONY: release
release: clean-release release_$(GOOS)


.PHONY: release_windows
release_windows: build
	cd $(OUT_DIR) ; zip -j ../$(NAME)_windows_$(GOARCH).zip  *


.PHONY: release_linux
release_linux: build
	cd $(OUT_DIR) ; tar -czf ../$(NAME)_linux_$(GOARCH).tar.gz *


.PHONY: clean-release
clean-release:
	$(RM) $(call FixPath,$(OUT_DIR))
