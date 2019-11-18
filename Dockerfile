###############################################################################
# BUILD STAGE

FROM golang:1.13-alpine

RUN set -x \
  && apk --no-cache --update add \
    bash \
    ca-certificates \
    curl \
    git \
    make \
    upx

COPY . /go/src/github.com/9seconds/mtg/

RUN set -x \
  && cd /go/src/github.com/9seconds/mtg \
  && make -j 4 static \
  && upx --ultra-brute -qq ./mtg


###############################################################################
# PACKAGE STAGE

FROM scratch

ENTRYPOINT ["/mtg"]
ENV MTG_IP=0.0.0.0 \
    MTG_PORT=3128 \
    MTG_STATS_IP=0.0.0.0 \
    MTG_STATS_PORT=3129
EXPOSE 3128 3129

COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /go/src/github.com/9seconds/mtg/mtg /mtg
