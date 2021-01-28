PKG := github.com/ExchangeUnion/xud-docker-api

GO_BIN := ${GOPATH}/bin

GOBUILD := go build -v

VERSION := local
COMMIT := $(shell git rev-parse HEAD)
ifeq ($(OS),Windows_NT)
	TIMESTAMP := $(shell powershell.exe scripts\get_timestamp.ps1)
else
	TIMESTAMP := $(shell date +%s)
endif
LDFLAGS := -ldflags "-w -s \
-X $(PKG)/build.Version=$(VERSION) \
-X $(PKG)/build.GitCommit=$(COMMIT) \
-X $(PKG)/build.Timestamp=$(TIMESTAMP)"

default: build


#
# Building
#

build:
	$(GOBUILD) $(LDFLAGS) ./cmd/proxy

.PHONY: build
