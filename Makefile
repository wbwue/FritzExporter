SHELL := /usr/bin/env bash
NAME := FritzExporter
IMPORT := github.com/wbwue/$(NAME)
BIN := bin
DIST := dist
GO := go
EXECUTABLE := $(NAME)

PACKAGES ?= $(shell go list ./...)
SOURCES ?= $(shell find . -name "*.go" -type f)
GENERATE ?= $(PACKAGES)

ifndef DATE
	DATE := $(shell date -u '+%Y%m%d')
endif

ifndef VERSION
	VERSION ?= $(shell git rev-parse --short HEAD)
endif

ifndef REVISION
	REVISION ?= $(shell git rev-parse --short HEAD)
endif

LDFLAGS += -s -w
LDFLAGS += -X "main.Version=$(VERSION)"
LDFLAGS += -X "main.BuildDate=$(DATE)"
LDFLAGS += -X "main.Revision=$(REVISION)"

.PHONY: all
all: build

.PHONY: clean
clean:
	$(GO) clean -i ./...
	rm -rf $(BIN)/
	rm -rf $(DIST)/

.PHONY: sync
sync:
	$(GO) mod download

.PHONY: fmt
fmt:
	$(GO) fmt $(PACKAGES)

.PHONY: vet
vet:
	$(GO) vet $(PACKAGES)

.PHONY: generate
generate:
	$(GO) generate $(GENERATE)

.PHONY: lint
lint:
	@which golangci-lint > /dev/null; if [ $$? -ne 0 ]; then \
		(echo "please install golangci-lint"; exit 1) \
	fi
	golangci-lint run -v

.PHONY: test
test:
	@which goverage > /dev/null; if [ $$? -ne 0 ]; then \
		GO111MODULE=off $(GO) get -u github.com/haya14busa/goverage; \
	fi
	goverage -v -coverprofile coverage.out $(PACKAGES)

.PHONY: build
build: $(BIN)/$(EXECUTABLE)

$(BIN)/$(EXECUTABLE): $(SOURCES)
	$(GO) build -i -v -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o $@ .

.PHONY: release
release: release-dirs release-build release-checksums

.PHONY: release-dirs
release-dirs:
	mkdir -p $(DIST)

.PHONY: release-build
release-build:
	@which gox > /dev/null; if [ $$? -ne 0 ]; then \
		GO111MODULE=off  $(GO) get -u github.com/mitchellh/gox; \
	fi
	gox  -os="linux darwin" -arch="amd64" -verbose -ldflags '-w $(LDFLAGS)' -output="$(DIST)/$(EXECUTABLE)-{{.OS}}-{{.Arch}}" .

.PHONY: release-checksums
release-checksums:
	cd $(DIST); $(foreach file, $(wildcard $(DIST)/$(EXECUTABLE)-*), sha256sum $(notdir $(file)) > $(notdir $(file)).sha256;)
