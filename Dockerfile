###############################################################################
# BUILD STAGE

FROM alpine:3 AS build

ENV CGO_ENABLED=0
ENV GOOS=linux

RUN set -x \
  && apk --no-cache --update add \
    bash \
    ca-certificates \
    git \
    mise

COPY . /app
WORKDIR /app

RUN set -x \
  && mise trust \
  && mise tasks run static


###############################################################################
# PACKAGE STAGE

FROM scratch

ENTRYPOINT ["/mtg"]
CMD ["run", "/config.toml"]

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /app/mtg /mtg
COPY --from=build /app/example.config.toml /config.toml
