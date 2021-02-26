###############################################################################
# BUILD STAGE

FROM golang:1.15-alpine AS build

RUN set -x \
  && apk --no-cache --update add \
    bash \
    ca-certificates \
    curl \
    git \
    make

COPY . /app
WORKDIR /app

RUN set -x \
  && make -j 4 static


###############################################################################
# PACKAGE STAGE

FROM scratch

ENTRYPOINT ["/mtg"]
ENV MTG_BIND=0.0.0.0:3128 \
    MTG_STATS_BIND=0.0.0.0:3129
EXPOSE 3128 3129

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /app/mtg /mtg
