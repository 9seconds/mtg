###############################################################################
# BUILD STAGE

FROM golang:1.26-alpine AS build

ENV CGO_ENABLED=0

# this is done for backward compatibility: before that we mounted a config
# into /config.toml. Some application allow mounting directories only,
# so it makes problems. So, instead we are going to do 2 steps:
#  1. Create /config/config.toml as a symlink to /config.toml
#  2. Force /mtg to use /config/config.toml
#
# it helps in both ways: users with directories could use /config directory
# and overlap a symlink by their bind mount. Old users could continue using
# /config.toml as a real config.
RUN set -x \
  && mkdir -p /config \
  && ln -sv /config.toml /config/config.toml

RUN --mount=type=cache,target=/var/cache/apk \
    set -x \
    && apk --update add \
      bash \
      ca-certificates \
      git

COPY go.mod go.sum /app/
WORKDIR /app

RUN go mod download

COPY . /app

RUN set -x \
  && version="$(git describe --exact-match HEAD 2>/dev/null || git describe --tags --always)" \
  && go build \
      -trimpath \
      -mod=readonly \
      -ldflags="-extldflags '-static' -s -w -X 'main.version=$version'" \
      -a \
      -tags netgo


###############################################################################
# PACKAGE STAGE

FROM scratch

ENTRYPOINT ["/mtg"]
CMD ["run", "/config/config.toml"]

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /app/mtg /mtg
COPY --from=build /app/example.config.toml /config.toml
COPY --from=build /config /config
