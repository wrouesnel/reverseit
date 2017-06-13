
COVERDIR = .coverage
TOOLDIR = tools

GO_SRC := $(shell find . -name '*.go' ! -path '*/vendor/*' ! -path 'tools/*' )
GO_DIRS := $(shell find . -type d -name '*.go' ! -path '*/vendor/*' ! -path 'tools/*' )
GO_PKGS := $(shell go list ./...)

VERSION ?= $(shell git describe --dirty)

CONCURRENT_LINTERS ?= $(shell cat /proc/cpuinfo | grep processor | wc -l)
LINTER_DEADLINE ?= 30s

export PATH := $(GOPATH)/bin:$(PATH)
SHELL := env PATH=$(PATH) /bin/bash

all: \
	style lint test reverseit

# Release builds the cross-platform named binaries we upload
release: reverseit-linux-arm64c \
			reverseit-linux-x86_64 \
			reverseit-linux-i386 \
			reverseit-windows-i386.exe \
			reverseit-windows-x86_64.exe \
			reverseit-darwin-x86_64 \
			reverseit-darwin-arm64 \
			reverseit-freebsd-x86_64

reverseit-linux-arm64c: $(GO_SRC)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a \
    	-ldflags "-extldflags '-static' -X main.Version=$(shell git describe --long --dirty)" \
    	-o $@ ./cmd/reverseit

reverseit-linux-x86_64: $(GO_SRC)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a \
    	-ldflags "-extldflags '-static' -X main.Version=$(shell git describe --long --dirty)" \
    	-o $@ ./cmd/reverseit

reverseit-linux-i386: $(GO_SRC)
	CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -a \
    	-ldflags "-extldflags '-static' -X main.Version=$(shell git describe --long --dirty)" \
    	-o $@ ./cmd/reverseit

reverseit-windows-i386.exe: $(GO_SRC)
	CGO_ENABLED=0 GOOS=windows GOARCH=386 go build -a \
    	-ldflags "-extldflags '-static' -X main.Version=$(shell git describe --long --dirty)" \
    	-o $@ ./cmd/reverseit

reverseit-windows-x86_64.exe: $(GO_SRC)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -a \
    	-ldflags "-extldflags '-static' -X main.Version=$(shell git describe --long --dirty)" \
    	-o $@ ./cmd/reverseit

reverseit-darwin-x86_64: $(GO_SRC)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -a \
    	-ldflags "-extldflags '-static' -X main.Version=$(shell git describe --long --dirty)" \
    	-o $@ ./cmd/reverseit

reverseit-darwin-arm64: $(GO_SRC)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -a \
    	-ldflags "-extldflags '-static' -X main.Version=$(shell git describe --long --dirty)" \
    	-o $@ ./cmd/reverseit

reverseit-freebsd-x86_64: $(GO_SRC)
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -a \
    	-ldflags "-extldflags '-static' -X main.Version=$(shell git describe --long --dirty)" \
    	-o $@ ./cmd/reverseit

reverseit: $(GO_SRC)
	CGO_ENABLED=0 go build -a \
    	-ldflags "-extldflags '-static' -X main.Version=$(shell git describe --long --dirty)" \
    	-o $@ ./cmd/reverseit

style: tools
	golangci-lint --concurrency $(CONCURRENT_LINTERS) run --disable-all --enable=gofmt --enable=goimports

lint: tools
	@echo Using $(CONCURRENT_LINTERS) processes
	golangci-lint --concurrency $(CONCURRENT_LINTERS) run

fmt: tools
	golangci-lint --concurrency $(CONCURRENT_LINTERS) run --disable-all --enable=gofmt --enable=goimports --fix

test: tools
	@mkdir -p $(COVERDIR)
	@rm -f $(COVERDIR)/*
	for pkg in $(GO_PKGS) ; do \
		go test -v -covermode count -coverprofile=$(COVERDIR)/$$(echo $$pkg | tr '/' '-').out $$pkg ; \
	done
	gocovmerge $(shell find $(COVERDIR) -name '*.out') > cover.out


tools:
	@echo Installing tools from tools.go
	@cd tools ; cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %

.PHONY: tools style fmt test release all
