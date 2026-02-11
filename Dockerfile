###############################################################################
# BUILD STAGE

FROM golang:1.26-alpine AS build

ENV CGO_ENABLED=0

RUN set -x \
  && apk --no-cache --update add \
    bash \
    ca-certificates \
    git

COPY . /app
WORKDIR /app

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
CMD ["run", "/config.toml"]

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /app/mtg /mtg
COPY --from=build /app/example.config.toml /config.toml
