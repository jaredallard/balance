# go option
GO          ?= go
PKG         := go mod vendor
LDFLAGS     := -w -s
GOFLAGS     :=
TAGS        := 
BINDIR      := $(CURDIR)/bin
PKGDIR      := github.com/jaredallard/balance
CGO_ENABLED := 1

# Required for globs to work correctly
SHELL=/bin/bash


.PHONY: all
all: build

.PHONY: dep
dep:
	@echo " ===> Installing dependencies via '$$(awk '{ print $$1 }' <<< "$(PKG)" )' <=== "
	@$(PKG)

.PHONY: build
build:
	@echo " ===> building releases in ./bin/... <=== "
	GO111MODULE=on CGO_ENABLED=$(CGO_ENABLED) $(GO) build -o $(BINDIR)/balance -v $(GOFLAGS) -tags '$(TAGS)' -ldflags '$(LDFLAGS)' $(PKGDIR)/cmd/...

.PHONY: gofmt
gofmt:
	@echo " ===> Running go fmt <==="
	goimports -w ./