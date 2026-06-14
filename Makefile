GO ?= $(shell if command -v go >/dev/null 2>&1; then command -v go; elif test -x "$(HOME)/.local/go/bin/go"; then echo "$(HOME)/.local/go/bin/go"; elif test -x /usr/local/go/bin/go; then echo /usr/local/go/bin/go; fi)
NPM ?= $(shell command -v npm 2> /dev/null)
MANIFEST_FILE ?= plugin.json
GOPATH ?= $(shell $(GO) env GOPATH 2>/dev/null)
GO_TEST_FLAGS ?= -race
GO_BUILD_FLAGS ?=
MM_UTILITIES_DIR ?= ../mattermost-utilities

export GO111MODULE=on

ASSETS_DIR ?= assets

include build/setup.mk

BUNDLE_NAME ?= $(PLUGIN_ID)-$(PLUGIN_VERSION).tar.gz

ifneq ($(wildcard build/custom.mk),)
	include build/custom.mk
endif

.PHONY: all apply check-style gofmt govet golint server webapp webapp-debug bundle dist deploy debug-deploy debug-dist test coverage clean help
all: check-style test dist

apply:
	./build/bin/manifest apply

check-style: webapp/.npminstall gofmt govet golint
ifneq ($(HAS_WEBAPP),)
	cd webapp && npm run lint
endif

gofmt:
ifneq ($(HAS_SERVER),)
	@echo Running gofmt
	@for package in $$(go list ./...); do \
		files=$$(go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}} {{end}}' $$package); \
		if [ "$$files" ]; then \
			gofmt_output=$$(gofmt -d -s $$files 2>&1); \
			if [ "$$gofmt_output" ]; then \
				echo "$$gofmt_output"; \
				exit 1; \
			fi; \
		fi; \
	done
endif

govet:
ifneq ($(HAS_SERVER),)
	$(GO) vet ./...
endif

golint:
	@echo Running lint
	env GO111MODULE=off $(GO) get golang.org/x/lint/golint
	$(GOPATH)/bin/golint -set_exit_status ./...
	@echo lint success

server:
ifneq ($(HAS_SERVER),)
	mkdir -p server/dist;
	cd server && env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -o dist/plugin-linux-amd64;
	cd server && env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -o dist/plugin-linux-arm64;
	cd server && env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -o dist/plugin-darwin-amd64;
	cd server && env CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -o dist/plugin-darwin-arm64;
	cd server && env CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(GO_BUILD_FLAGS) -trimpath -o dist/plugin-windows-amd64.exe;
endif

webapp/.npminstall:
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) install
	touch $@
endif

webapp: webapp/.npminstall
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run build;
endif

webapp-debug: webapp/.npminstall
ifneq ($(HAS_WEBAPP),)
	cd webapp && $(NPM) run debug;
endif

bundle:
	rm -rf dist/
	mkdir -p dist/$(PLUGIN_ID)
	cp $(MANIFEST_FILE) dist/$(PLUGIN_ID)/
ifneq ($(wildcard $(ASSETS_DIR)/.),)
	cp -r $(ASSETS_DIR) dist/$(PLUGIN_ID)/
endif
ifneq ($(HAS_PUBLIC),)
	cp -r public dist/$(PLUGIN_ID)/
endif
ifneq ($(HAS_SERVER),)
	mkdir -p dist/$(PLUGIN_ID)/server/dist;
	cp -r server/dist/* dist/$(PLUGIN_ID)/server/dist/;
endif
ifneq ($(HAS_WEBAPP),)
	mkdir -p dist/$(PLUGIN_ID)/webapp/dist;
	cp -r webapp/dist/* dist/$(PLUGIN_ID)/webapp/dist/;
endif
	cd dist && tar -cvzf $(BUNDLE_NAME) $(PLUGIN_ID)
	@echo plugin built at: dist/$(BUNDLE_NAME)

dist: apply server webapp bundle

deploy: dist
	./build/bin/pluginctl deploy $(PLUGIN_ID) dist/$(BUNDLE_NAME)

debug-dist: apply server webapp-debug bundle

test: webapp/.npminstall
ifneq ($(HAS_SERVER),)
	$(GO) test -v $(GO_TEST_FLAGS) ./server/...
endif

clean:
	rm -fr dist/
ifneq ($(HAS_SERVER),)
	rm -fr server/dist
endif
ifneq ($(HAS_WEBAPP),)
	rm -fr webapp/.npminstall webapp/dist webapp/node_modules
endif
	rm -fr build/bin/

help:
	@cat Makefile | grep -v '\.PHONY' | grep -v '\help:' | grep -B1 -E '^[a-zA-Z0-9_.-]+:.*' | sed -e "s/:.*//" | sed -e "s/^## //" | grep -v '\-\-' | sed '1!G;h;$$!d' | awk 'NR%2{printf "\033[36m%-30s\033[0m",$$0;next;}1' | sort
