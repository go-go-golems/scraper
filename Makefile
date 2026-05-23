.PHONY: all test test-go test-web build build-go build-web generate proto lint lintmax docker-lint golangci-lint-install gosec govulncheck goreleaser tag-major tag-minor tag-patch release install version clean dev-up dev-down dev-status dev-logs validate tidy

all: test build

BINARY ?= scraper
VERSION ?= $(shell svu current 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS ?= -X main.version=$(VERSION)
GORELEASER_ARGS ?= --skip=sign --snapshot --clean
GORELEASER_TARGET ?= --single-target
GOLANGCI_LINT_VERSION ?= $(shell cat .golangci-lint-version)
GOLANGCI_LINT_BIN ?= $(CURDIR)/.bin/golangci-lint
GO_PACKAGES ?= ./...
WEB_DIR ?= web
DIST_DIR ?= dist

version:
	@echo $(VERSION)

generate proto:
	buf generate

tidy:
	go mod tidy

validate: test build

test: test-go test-web

test-go:
	go test $(GO_PACKAGES) -count=1

test-web:
	cd $(WEB_DIR) && pnpm test:unit -- --runInBand

build: build-go build-web

build-go:
	go generate ./...
	mkdir -p $(DIST_DIR)
	go build -tags "sqlite_fts5 embed" -ldflags "$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY) ./cmd/$(BINARY)

build-web:
	cd $(WEB_DIR) && pnpm build

docker-lint:
	docker run --rm -v $(CURDIR):/app -w /app golangci/golangci-lint:$(GOLANGCI_LINT_VERSION) golangci-lint run -v ./cmd/... ./pkg/...

golangci-lint-install:
	mkdir -p $(dir $(GOLANGCI_LINT_BIN))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(dir $(GOLANGCI_LINT_BIN)) $(GOLANGCI_LINT_VERSION)

lint: golangci-lint-install
	$(GOLANGCI_LINT_BIN) run -v ./cmd/... ./pkg/...

lintmax: golangci-lint-install
	$(GOLANGCI_LINT_BIN) run -v --max-same-issues=100 ./cmd/... ./pkg/...

gosec:
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude=G101,G304,G301,G306,G204 -exclude-dir=ttmp -exclude-dir=.history -exclude-dir=web/node_modules ./...

govulncheck:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

goreleaser:
	GOWORK=off goreleaser release $(GORELEASER_ARGS) $(GORELEASER_TARGET)

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

release:
	git push origin --tags
	GOPROXY=proxy.golang.org go list -m github.com/go-go-golems/scraper@$(shell svu current)

install: build-go
	cp $(DIST_DIR)/$(BINARY) $(shell go env GOPATH)/bin/$(BINARY)

clean:
	rm -rf $(DIST_DIR)
	rm -rf $(WEB_DIR)/dist
	rm -rf .bin

# Local development stack. Requires devctl and Docker for Redis.
dev-up:
	devctl up

dev-down:
	devctl down

dev-status:
	devctl status --tail-lines 10

dev-logs:
	devctl logs --service api --follow
