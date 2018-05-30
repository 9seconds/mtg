###############################################################################
# BUILD STAGE

FROM golang:alpine

RUN set -x \
  && apk --no-cache --update add \
    bash \
    ca-certificates \
    curl \
    git \
    make \
  && update-ca-certificates

ADD . /go/src/github.com/9seconds/mtg

RUN set -x \
  && cd /go/src/github.com/9seconds/mtg \
  && make clean \
  && make -j 4 static


###############################################################################
# PACKAGE STAGE

FROM scratch

ENTRYPOINT ["/usr/local/bin/mtg"]
ENV MTG_IP=0.0.0.0 \
    MTG_PORT=3128 \
    MTG_STATS_IP=0.0.0.0 \
    MTG_STATS_PORT=3130
EXPOSE 3128 3130

COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=0 /go/src/github.com/9seconds/mtg /usr/local/bin/mtg
