APP     := mqtt-manager
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

.DEFAULT_GOAL := help

.PHONY: help dev build build-debug run clean test tidy version changelog \
        release release-patch release-minor release-major

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

version: ## Print the version that would be built
	@echo $(VERSION)

dev: ## Run in hot-reload dev mode
	wails dev

build: ## Build the production app bundle (version injected from git)
	wails build -ldflags "$(LDFLAGS)"
	@scripts/macicon.sh

build-debug: ## Build with devtools enabled
	wails build -debug -devtools -ldflags "$(LDFLAGS)"
	@scripts/macicon.sh

dist: ## Build a universal macOS .app and zip it for sharing
	wails build -platform darwin/universal -ldflags "$(LDFLAGS)"
	@scripts/macicon.sh
	@rm -f "build/bin/$(APP)-$(VERSION)-macos-universal.zip"
	@ditto -c -k --keepParent "build/bin/$(APP).app" "build/bin/$(APP)-$(VERSION)-macos-universal.zip"
	@echo "Wrote build/bin/$(APP)-$(VERSION)-macos-universal.zip"

run: build ## Build then launch the app (macOS)
	open build/bin/$(APP).app

clean: ## Remove build output
	rm -rf build/bin

test: ## Run Go tests
	go test ./...

tidy: ## Tidy Go modules
	go mod tidy

changelog: ## Preview unreleased changes since the last tag
	@scripts/release.sh changelog

release: ## Cut a release (defaults to a patch bump; override with V=1.2.3)
	@scripts/release.sh release "$(V)"

release-patch: ## Cut a patch release (x.y.Z)
	@scripts/release.sh release patch

release-minor: ## Cut a minor release (x.Y.0)
	@scripts/release.sh release minor

release-major: ## Cut a major release (X.0.0)
	@scripts/release.sh release major
