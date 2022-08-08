ROOT_DIR     := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
IMAGE_NAME   := mtg
APP_NAME     := $(IMAGE_NAME)

GOLANGCI_LINT_VERSION := v1.48.0

VERSION            := $(shell git describe --exact-match HEAD 2>/dev/null || git describe --tags --always)
COMMON_BUILD_FLAGS := -trimpath -mod=readonly -ldflags="-extldflags '-static' -s -w -X 'main.version=$(VERSION)'"

FUZZ_FLAGS := -fuzztime=120s

GOBIN  := $(ROOT_DIR)/.bin
GOTOOL := env "GOBIN=$(GOBIN)" "PATH=$(ROOT_DIR)/.bin:$(PATH)"

# -----------------------------------------------------------------------------

.PHONY: all
all: build

.PHONY: build
build:
	@go build $(COMMON_BUILD_FLAGS) -o "$(APP_NAME)"

$(APP_NAME): build

.PHONY: static
static:
	@env CGO_ENABLED=0 GOOS=linux go build \
		$(COMMON_BUILD_FLAGS) \
		-tags netgo \
		-a \
		-o "$(APP_NAME)"

vendor: go.mod go.sum
	@$(MOD_ON) go mod vendor

.bin:
	@mkdir -p "$(GOBIN)" || true

.PHONY: fmt
fmt:
	@$(GOTOOL) gofumpt -w -extra "$(ROOT_DIR)"

.PHONY: test
test:
	@go test -v ./...

.PHONY: citest
citest:
	@go test -coverprofile=coverage.txt -covermode=atomic -parallel 2 -race -v ./...

.PHONY: clean
clean:
	@git clean -xfd && \
		git reset --hard >/dev/null && \
		git submodule foreach --recursive sh -c 'git clean -xfd && git reset --hard' >/dev/null

.PHONY: lint
lint:
	@$(GOTOOL) golangci-lint run

.PHONY: release
release:
	@$(GOTOOL) goreleaser release --snapshot --rm-dist && \
		find "$(ROOT_DIR)/dist" -type d | grep -vP "dist$$" | xargs -r rm -rf && \
		rm -f "$(ROOT_DIR)/dist/config.yaml"

.PHONY: docker
docker:
	@docker build --pull -t "$(IMAGE_NAME)" "$(ROOT_DIR)"

.PHONY: doc
doc:
	@$(GOTOOL) godoc -http 0.0.0.0:10000

.PHONY: install-tools
install-tools: install-tools-lint install-tools-godoc install-tools-gofumpt install-tools-goreleaser

.PHONY: install-tools-lint
install-tools-lint: .bin
	@curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
		| bash -s -- -b "$(GOBIN)" "$(GOLANGCI_LINT_VERSION)"

.PHONY: install-tools-godoc
install-tools-godoc: .bin
	@$(GOTOOL) go install golang.org/x/tools/cmd/godoc@latest

.PHONY: install-tools-gofumpt
install-tools-gofumpt: .bin
	@$(GOTOOL) go install mvdan.cc/gofumpt@latest

.PHONY: goreleaser
install-tools-goreleaser: .bin
	@$(GOTOOL) go install github.com/goreleaser/goreleaser@latest

.PHONY: update-deps
update-deps:
	@go get -u && go mod tidy -go=1.18

.PHONY: fuzz
fuzz: fuzz-ClientHello fuzz-ServerGenerateHandshakeFrame fuzz-ClientHandshake fuzz-ServerReceive fuzz-ServerSend

.PHONY: fuzz-ClientHello
fuzz-ClientHello:
	@go test -fuzz=FuzzClientHello $(FUZZ_FLAGS) "$(ROOT_DIR)/mtglib/internal/faketls"

.PHONY: fuzz-ServerGenerateHandshakeFrame
fuzz-ServerGenerateHandshakeFrame:
	@go test -fuzz=FuzzServerGenerateHandshakeFrame $(FUZZ_FLAGS) "$(ROOT_DIR)/mtglib/internal/obfuscated2"

.PHONY: fuzz-ClientHandshake
fuzz-ClientHandshake:
	@go test -fuzz=FuzzClientHandshake $(FUZZ_FLAGS) "$(ROOT_DIR)/mtglib/internal/obfuscated2"

.PHONY: fuzz-ServerReceive
fuzz-ServerReceive:
	@go test -fuzz=FuzzServerReceive $(FUZZ_FLAGS) "$(ROOT_DIR)/mtglib/internal/obfuscated2"

.PHONY: fuzz-ServerSend
fuzz-ServerSend:
	@go test -fuzz=FuzzServerSend $(FUZZ_FLAGS) "$(ROOT_DIR)/mtglib/internal/obfuscated2"
