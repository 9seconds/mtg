ROOT_DIR     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
IMAGE_NAME   := mtg
APP_NAME     := $(IMAGE_NAME)

VENDOR_FILES := $(shell find "$(ROOT_DIR)/vendor" 2>/dev/null || echo -n "vendor")
CC_BINARIES  := $(shell bash -c "echo -n $(APP_NAME)-{linux,freebsd,openbsd}-{386,amd64} $(APP_NAME)-linux-{arm,arm64}")
APP_DEPS     := version.go $(VENDOR_FILES)

GOLANGCI_LINT_VERSION := v1.9.1

COMMON_BUILD_FLAGS := -ldflags="-s -w"

# -----------------------------------------------------------------------------

$(APP_NAME): $(APP_DEPS)
	@go build $(COMMON_BUILD_FLAGS) -o "$(APP_NAME)"

static-$(APP_NAME): $(APP_DEPS)
	@env CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo $(COMMON_BUILD_FLAGS) -o "$(APP_NAME)"

$(APP_NAME)-%: GOOS=$(shell echo -n "$@" | sed 's?$(APP_NAME)-??' | cut -f1 -d-)
$(APP_NAME)-%: GOARCH=$(shell echo -n "$@" | sed 's?$(APP_NAME)-??' | cut -f2 -d-)
$(APP_NAME)-%: $(APP_DEPS) ccbuilds
	@env "GOOS=$(GOOS)" "GOARCH=$(GOARCH)" \
		go build \
		$(COMMON_BUILD_FLAGS) \
		-o "./ccbuilds/$(APP_NAME)-$(GOOS)-$(GOARCH)"

ccbuilds:
	@rm -rf ./ccbuilds && mkdir -p ./ccbuilds

version.go:
	@go generate main.go

vendor: Gopkg.lock Gopkg.toml install-dep
	@dep ensure

# -----------------------------------------------------------------------------

.PHONY: all
all: $(APP_NAME)

.PHONY: static
static: static-$(APP_NAME)

.PHONY: crosscompile
crosscompile: $(CC_BINARIES)

.PHONY: crosscompile-dir
crosscompile-dir:
	@rm -rf "$(CC_DIR)" && mkdir -p "$(CC_DIR)"

.PHONY: test
test: vendor version.go
	@go test -v ./...

.PHONY: lint
lint: version.go
	@golangci-lint run

.PHONY: critic
critic: version.go
	@gocritic check-project "$(ROOT_DIR)"

.PHONY: clean
clean:
	@git clean -xfd && \
		git reset --hard >/dev/null && \
		git submodule foreach --recursive sh -c 'git clean -xfd && git reset --hard' >/dev/null

.PHONY: docker
docker:
	@docker build --pull -t "$(IMAGE_NAME)" "$(ROOT_DIR)"

.PHONY: prepare
prepare: install-dep install-lint install-critic

.PHONY: install-dep
install-dep:
	@go get -u github.com/golang/dep/cmd/dep

.PHONY: install-lint
install-lint:
	@curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh \
		| bash -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION)

.PHONY: install-critic
install-critic:
	@go get -u github.com/go-critic/go-critic/...
