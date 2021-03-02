ROOT_DIR     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
IMAGE_NAME   := mtg
APP_NAME     := $(IMAGE_NAME)

CC_BINARIES  := $(shell bash -c "echo -n $(APP_NAME)-{linux,freebsd,openbsd}-{386,amd64} $(APP_NAME)-linux-{arm,arm64}")

GOLANGCI_LINT_VERSION := v1.37.1

VERSION_GO         := $(shell go version)
VERSION_DATE       := $(shell date -Ru)
VERSION_TAG        := $(shell git describe --tags --always)
COMMON_BUILD_FLAGS := -mod=readonly -ldflags="-s -w -X 'main.version=$(VERSION_TAG) ($(VERSION_GO)) [$(VERSION_DATE)]'"

GOBIN  := $(ROOT_DIR)/.bin
GOTOOL := env "GOBIN=$(GOBIN)" "PATH=$(ROOT_DIR)/.bin:$(PATH)"

# -----------------------------------------------------------------------------

.PHONY: all
all: build

.PHONY: build
build: $(APP_NAME)

$(APP_NAME):
	@go build $(COMMON_BUILD_FLAGS) -o "$(APP_NAME)"

static-$(APP_NAME):
	@env CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo $(COMMON_BUILD_FLAGS) -o "$(APP_NAME)"

$(APP_NAME)-%: GOOS=$(shell echo -n "$@" | sed 's?$(APP_NAME)-??' | cut -f1 -d-)
$(APP_NAME)-%: GOARCH=$(shell echo -n "$@" | sed 's?$(APP_NAME)-??' | cut -f2 -d-)
$(APP_NAME)-%: ccbuilds
	@env "GOOS=$(GOOS)" "GOARCH=$(GOARCH)" \
		go build \
		$(COMMON_BUILD_FLAGS) \
		-o "./ccbuilds/$(APP_NAME)-$(GOOS)-$(GOARCH)"

ccbuilds:
	@rm -rf ./ccbuilds && mkdir -p ./ccbuilds

vendor: go.mod go.sum
	@$(MOD_ON) go mod vendor

.PHONY: test
test:
	@go test -v ./...

.PHONY: citest
citest:
	@go test -coverprofile=coverage.txt -covermode=atomic -race -v ./...

.PHONY: crosscompile
crosscompile: $(CC_BINARIES)

.PHONY: clean
clean:
	@git clean -xfd && \
		git reset --hard >/dev/null && \
		git submodule foreach --recursive sh -c 'git clean -xfd && git reset --hard' >/dev/null

.PHONY: lint
lint:
	@$(GOTOOL) golangci-lint run

.PHONY: docker
docker:
	@docker build --pull -t "$(IMAGE_NAME)" "$(ROOT_DIR)"

.PHONY: doc
doc:
	@$(GOTOOL) godoc -http 0.0.0.0:10000

.PHONY: install-tools
install-tools: install-tools-lint install-tools-godoc

.PHONY: install-tools-lint
install-tools-lint:
	@mkdir -p "$(GOBIN)" || true && \
		curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh \
		| bash -s -- -b "$(GOBIN)" "$(GOLANGCI_LINT_VERSION)"

.PHONY: install-tools-godoc
install-tools-godoc:
	@mkdir -p "$(GOBIN)" || true && \
		$(GOTOOL) go get -u golang.org/x/tools/cmd/godoc

.PHONY: update-deps
upgrade-deps:
	$go get -u && go mod tidy
