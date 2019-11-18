# mtg

Bullshit-free MTPROTO proxy for Telegram

[![Build Status](https://travis-ci.org/9seconds/mtg.svg?branch=master)](https://travis-ci.org/9seconds/mtg)
[![Go Report Card](https://goreportcard.com/badge/github.com/9seconds/mtg)](https://goreportcard.com/report/github.com/9seconds/mtg)
[![Docker Build Status](https://img.shields.io/docker/build/nineseconds/mtg.svg)](https://hub.docker.com/r/nineseconds/mtg/)

**Please see a guide on upgrading to 1.0 at the end of this README.**

# Rationale

There are several available proxies for Telegram MTPROTO available. Here
are the most notable:

* [Official](https://github.com/TelegramMessenger/MTProxy)
* [Python](https://github.com/alexbers/mtprotoproxy)
* [Erlang](https://github.com/seriyps/mtproto_proxy)

Almost all of them follow the way how official proxy was built. This
includes support of multiple secrets, support of promoted channels, etc.

mtg is an implementation in golang which is intended to be:

* **Lightweight**
  It has to consume as few resources as possible but not by losing
  maintainability.
* **Easily deployable**
  I strongly believe that Telegram proxies should follow the way of
  ShadowSocks: promoted channels is a strange way of doing business
  I suppose. I think the only viable way is to have a proxy with
  minimum configuration which should work everywhere.
* **A single secret**
  I think that multiple secrets solve no problems and just complexify
  software. I also believe that in the case of throwout proxies, this
  feature is a useless luxury.
* **Minimum docker image size**
  Official image is less than 3.5 megabytes. Literally.
* **No management WebUI**
  This is an implementation of a simple lightweight proxy. I won't do that.

This proxy supports 2 modes of work: direct connection to Telegram and
promoted channel mode. If you do not need promoted channels, I would
recommend you to go with direct mode: this way is more robust.

To run a proxy in direct mode, all you need to do is just provide a
secret. If you do not provide ADTag as a second parameter, promoted
channels mode won't be activated.

To get promoted channel, please contact
[@MTProxybot](https://t.me/MTProxybot) and provide generated adtag as a
second parameter.


# Source code organization

There are 2 main branches:

1. `master` branch contains potentially unstable features
2. `stable` branch contains stable version. Usually you want to use this branch.

# How to build

```console
make
```

If you want to build for another platform:

```console
make crosscompile
```

If you want to build Docker image (called `mtg`):

```console
make docker
```

# Docker image

Docker follows the same policy as the source code organization:

- `latest` mirrors the master branch
- `stable` mirrors the stable branch
- tags are for tagged releases

```console
docker pull nineseconds/mtg:latest
```

```console
docker pull nineseconds/mtg:stable
```

```console
docker pull nineseconds/mtg:0.10
```

# Ansible role

You can find unofficial Ansible role for mtg here: https://github.com/rlex/ansible-role-mtg
Also, there is another project on Ansible Galaxy: https://galaxy.ansible.com/ivansible/lin_mtproxy

# Configuration

To run this tool you need to configure as less as possible. Telegram
clients support 3 different secret types:

* Simple - basically, it is just a flow of frames ciphered by AES-CTR stream
  cipher.
* Secured - the same stream as simple but with some random noise to prevent
  statistical analysis of traffic flow.
* FakeTLS - this mode envelops telegram stream in TLS so it looks (in theory)
  the same as any TLS1.3 traffic from DPI point of view.

If you do not have preferences, go with FakeTLS or at least secured.
Simple mode is a little bit naive and traffic flow can be easily
identified as Telegram one.

Unlike the rest of implementation, mtg is quite strict about the
execution mode: if you run a proxy instance with FakeTLS secret, you
can't connect to it with simple or secured clients. You can't connect
to the proxy with secured secret with FakeTLS key. It forces one mode
of working. So, unfortunately, there is no way how to connect to the
deployed proxy with another secret (if you know how to construct and
convert them). But at the same time, old clients can't connect so they
won't expose the type of the service.

First, you need to generate a secret:

```console
$ mtg generate-secret simple
52a493bdfb90eea55739eabff2d92a14
```

```console
$ mtg generate-secret secured
ddf05fb7acb549be047a7c585116581418
```

```console
$ mtg generate-secret -c google.com tls
ee852380f362a09343efb4690c4e17862e676f6f676c652e636f6d
```

## Antireplay cache

To prevent replay attacks, we have internal storage of first frames
messages for connected clients. These frames are generated randomly
by design and we have the negligible possibility of duplication
(probability is 1/(2^64)) but it could be quite effective to prevent
replays.


## FakeTLS

If you run this a proxy in faketls mode, this proxy will try to hide
itself cloaking a host provided as a part of the generated secret. It
means that if you cloak google.com then you can curl this proxy and
you'll get a google.com response back.

mtg proxies L3 traffic. In other words, only TCP, without interfering in
TLS, HTTP or any other high-level protocol.


## Environment variables

It is possible to configure this tool using environment variables. You
can configure any flag but not secret or adtag. Here is the list of
supported environment variables:

| Environment variable          | Corresponding flags          | Default value                     | Description                                                                                                                                                                                                                                                                     |
|-------------------------------|------------------------------|-----------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `MTG_DEBUG`                   | `-d`, `--debug`              | `false`                           | Run in debug mode. Usually, you need to run in this mode  only if you develop this tool or its maintainer is asking you to provide  logs with such verbosity.                                                                                                                   |
| `MTG_VERBOSE`                 | `-v`, `--verbose`            | `false`                           | Run in verbose mode. This is way less chatty than debug mode.                                                                                                                                                                                                                   |
| `MTG_BIND`                    | `-b`, `--bind`               | `0.0.0.0:3128`                    | Which host/port pair should we bind to (listen on).                                                                                                                                                                                                                             |
| `MTG_IPV4`                    | `-4`, `--public-ipv4`        | [Autodetect](https://ifconfig.co) | IPv4 address:port of this proxy. This is required if you NAT your proxy or run it in a docker container. In that case, you absolutely need to specify public IPv4 address of the proxy, otherwise either URLs will be broken or proxy could not access Telegram middle proxies. |
| `MTG_IPV6`                    | `-6`, `--public-ipv6`        | [Autodetect](https://ifconfig.co) | IPv6 address:port of this proxy. This is required if you NAT your proxy or run it in a docker container. In that case, you absolutely need to specify public IPv6 address of the proxy, otherwise either URLs will be broken or proxy could not access Telegram middle proxies. |
| `MTG_STATS_BIND`              | `-t`, `--stats-bind`         | `127.0.0.1:3129`                  | Which hist:port should we bind the internal statistics HTTP server (Prometheus).                                                                                                                                                                                                |
| `MTG_STATS_NAMESPACE`         | `--stats-namespace`          | `mtg`                             | Which namespace should be used for prometheus metrics.                                                                                                                                                                                                                          |
| `MTG_STATSD_ADDR`             | `--statsd-addr`              |                                   | IP:host addresses of statsd service. No defaults, by defaults we do not send anything there.                                                                                                                                                                                    |
| `MTG_STATSD_PORT`             | `--statsd-port`              | `8125`                            | Which port should we use to work with statsd.                                                                                                                                                                                                                                   |
| `MTG_STATSD_PREFIX`           | `--statsd-prefix`            | `mtg`                             | Which bucket prefix we should use. For example, if you set `mtg`, then metric `traffic.ingress` would be send as `mtg.traffic.ingress`.                                                                                                                                         |
| `MTG_STATSD_TAGS_FORMAT`      | `--statsd-tags-format`       |                                   | Which tags format we should use. By default, we are using default vanilla statsd tags format but if you want to send directly to InfluxDB or Datadog, please specify it there. Possible options are `influxdb` and `datadog`.                                                   |
| `MTG_STATSD_TAGS`             | `--statsd-tags`              |                                   | Which tags should we send to statsd with our metrics. Please specify them as `key=value` pairs.                                                                                                                                                                                 |
| `MTG_BUFFER_WRITE`            | `-w`, `--write-buffer`       | `65536`                           | The size of TCP write buffer in bytes. Write buffer is the buffer for messages which are going from client to Telegram.                                                                                                                                                         |
| `MTG_BUFFER_READ`             | `-r`, `--read-buffer`        | `131072`                          | The size of TCP read buffer in bytes. Read buffer is the buffer for messages from Telegram to client.                                                                                                                                                                           |
| `MTG_ANTIREPLAY_MAXSIZE`      | `--anti-replay-max-size`     | `128MB`                           | Max size of antireplay cache.                                                                                                                                                                                                                                                   |
| `MTG_CLOAK_PORT`              | `--cloak-port`               | `443`                             | Which port we should use to connect to cloaked host in FakeTLS mode.                                                                                                                                                                                                            |
| `MTG_MULTIPLEX_PERCONNECTION` | `--multiplex-per-connection` | `50`                              | How many client connections can share a single Telegram connection in adtag mode                                                                                                                                                                                                |

Usually you want to modify only read/write buffer sizes. If you feel
that proxy is slow, try to increase both sizes giving more priority to
read buffer.

Unfortunately, MTPROTO proxy protocol does not allow us to use splice
or any other neat tricks how to eliminate the need of copying data into
userspace.

# How to run the tool

Now run the tool:

```console
$ mtg run <secret>
```

How to run the tool with ADTag:

```console
$ mtg run <secret> <adtag>
```

This tool will listen on port 3128 by default with the given secret.


# statsd integration

mtg provides an integration with statsd, you can enable it with command
line interface. To enable it, you have to provide IP address of statsd
service.

Out of the box, mtg supports 2 additional dialects: [InfluxDB](https://www.influxdata.com/blog/getting-started-with-sending-statsd-metrics-to-telegraf-influxdb/)
and [Datadog](https://docs.datadoghq.com/developers/dogstatsd/).

All metrics are gauges. Here is the list of metrics and their meaning:

| Metric name            | Unit    | Description                                |
|------------------------|---------|--------------------------------------------|
| `connections`          | number  | The number of active connections.          |
| `telegram_connections` | number  | The number of active telegram connections. |
| `crashes`              | number  | An amount of crashes in client handlers.   |
| `traffic.egress`       | bytes   | Traffic from the start of application.     |
| `replay_attacks`       | number  | The number of prevented replay attacks.    |

All metrics are prefixed with given prefix. Default prefix is `mtg`.
Also, metrics provide tags (ipv4/ipv6, dc indexes etc).


# Prometheus integration

[Prometheus](https://prometheus.io) integration comes out of
the box, you do not need to setup anything special.


# Upgrade to 1.0

Version 1.0 breaks compatibility with previous versions so please read
this chapter carefully:

1. mtg now uses subcommands. Please use `mtg run` instead of just
   `mtg` to run a proxy.
2. Options which set host and port separately were removed in a
   favor of fused `host:port` options.
3. Own stats server was removed. Prometheus endpoint is moved to
   default stats endpoint.
4. It is possible to connect to this proxy only with a secret which
   was used to run it. So, no backward compatibility of clients.
5. Multiplexing involves connectivity with middle proxies and involves
   the most complex code path of this proxy. To avoid potential bugs,
   we still recommend using direct mode.
