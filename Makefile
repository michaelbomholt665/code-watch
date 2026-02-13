.PHONY: test test-offline

CACHE_ROOT := $(CURDIR)/.cache
GO_BUILD_CACHE := $(CACHE_ROOT)/go-build
GO_TMP := $(CACHE_ROOT)/go-tmp
GO_MOD_CACHE := $(CACHE_ROOT)/go-mod

test:
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) GOTMPDIR=$(GO_TMP) GOMODCACHE=$(GO_MOD_CACHE) go test ./...

test-offline:
	mkdir -p $(GO_BUILD_CACHE) $(GO_TMP) $(GO_MOD_CACHE)
	GOCACHE=$(GO_BUILD_CACHE) GOTMPDIR=$(GO_TMP) GOMODCACHE=$(GO_MOD_CACHE) GOPROXY=off GOSUMDB=off GOFLAGS=-mod=vendor go test ./...
