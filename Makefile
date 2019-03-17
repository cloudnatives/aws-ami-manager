DEV_IMAGE         ?= golang:1.12
SHORT_NAME        ?= tiller
TARGETS           ?= darwin/amd64 linux/amd64 linux/386 linux/arm linux/arm64 windows/amd64
TARGET_OBJS       ?= darwin-amd64.tar.gz darwin-amd64.tar.gz.sha256 linux-amd64.tar.gz linux-amd64.tar.gz.sha256 linux-386.tar.gz linux-386.tar.gz.sha256 linux-arm.tar.gz linux-arm.tar.gz.sha256 linux-arm64.tar.gz linux-arm64.tar.gz.sha256 windows-amd64.zip windows-amd64.zip.sha256
DIST_DIRS         = find * -type d -exec

# go option
GO        ?= go
TAGS      :=
LDFLAGS   := -w -s
GOFLAGS   :=
BINDIR    := $(CURDIR)/bin

# Required for globs to work correctly
SHELL=/usr/bin/env bash

.PHONY: all
all: build

.PHONY: build
build:
	GOBIN=$(BINDIR) $(GO) install $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' .

# usage: make clean build-cross dist VERSION=v2.0.0-alpha.3
.PHONY: build-cross
build-cross: LDFLAGS += -extldflags "-static"
build-cross:
	CGO_ENABLED=0 gox -parallel=3 -output="_dist/{{.OS}}-{{.Arch}}/{{.Dir}}" -osarch='$(TARGETS)' $(GOFLAGS) $(if $(TAGS),-tags '$(TAGS)',) -ldflags '$(LDFLAGS)' .

.PHONY: dist
dist:
	( \
		cd _dist && \
		$(DIST_DIRS) cp ../LICENSE {} \; && \
		$(DIST_DIRS) cp ../README.md {} \; && \
		$(DIST_DIRS) tar -zcf aws-ami-manager-${VERSION}-{}.tar.gz {} \; && \
		$(DIST_DIRS) zip -r aws-ami-manager-${VERSION}-{}.zip {} \; \
	)

.PHONY: clean
clean:
	@rm -rf $(BINDIR) ./_dist