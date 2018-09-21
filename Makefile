ROOT_DIR     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
IMAGE_NAME   := mtg
APP_NAME     := $(IMAGE_NAME)

CC_BINARIES  := $(shell bash -c "echo -n $(APP_NAME)-{linux,freebsd,openbsd}-{386,amd64} $(APP_NAME)-linux-{arm,arm64}")
APP_DEPS     := version.go

GOLANGCI_LINT_VERSION := v1.10.2

COMMON_BUILD_FLAGS := -ldflags="-s -w"

MOD_ON  := env GO111MODULE=on
MOD_OFF := env GO111MODULE=auto

# -----------------------------------------------------------------------------

$(APP_NAME): $(APP_DEPS)
	@$(MOD_ON) go build $(COMMON_BUILD_FLAGS) -o "$(APP_NAME)"

static-$(APP_NAME): $(APP_DEPS)
	@$(MOD_ON) env CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo $(COMMON_BUILD_FLAGS) -o "$(APP_NAME)"

$(APP_NAME)-%: GOOS=$(shell echo -n "$@" | sed 's?$(APP_NAME)-??' | cut -f1 -d-)
$(APP_NAME)-%: GOARCH=$(shell echo -n "$@" | sed 's?$(APP_NAME)-??' | cut -f2 -d-)
$(APP_NAME)-%: $(APP_DEPS) ccbuilds
	@$(MOD_ON) env "GOOS=$(GOOS)" "GOARCH=$(GOARCH)" \
		go build \
		$(COMMON_BUILD_FLAGS) \
		-o "./ccbuilds/$(APP_NAME)-$(GOOS)-$(GOARCH)"

ccbuilds:
	@rm -rf ./ccbuilds && mkdir -p ./ccbuilds

version.go:
	@$(MOD_ON) go generate main.go

vendor: go.mod go.sum
	@$(MOD_ON) go mod vendor

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
test: vendor $(APP_DEPS)
	@$(MOD_ON) go test -v ./...

.PHONY: lint
lint: vendor $(APP_DEPS)
	@$(MOD_OFF) golangci-lint run

.PHONY: critic
critic: vendor $(APP_DEPS)
	@$(MOD_OFF) gocritic check-project "$(ROOT_DIR)"

.PHONY: clean
clean:
	@git clean -xfd && \
		git reset --hard >/dev/null && \
		git submodule foreach --recursive sh -c 'git clean -xfd && git reset --hard' >/dev/null

.PHONY: docker
docker:
	@docker build --pull -t "$(IMAGE_NAME)" "$(ROOT_DIR)"

.PHONY: prepare
prepare: install-lint install-critic

.PHONY: install-lint
install-lint:
	@curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh \
		| $(MOD_OFF) bash -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT_VERSION)

.PHONY: install-critic
install-critic:
	@$(MOD_OFF) go get -u github.com/go-critic/go-critic/...
