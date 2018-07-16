# mtg

Bullshit-free MTPROTO proxy for Telegram

[![Build Status](https://travis-ci.org/9seconds/mtg.svg?branch=master)](https://travis-ci.org/9seconds/mtg)
[![Docker Build Status](https://img.shields.io/docker/build/nineseconds/mtg.svg)](https://hub.docker.com/r/nineseconds/mtg/)


# Rationale

There are several available proxies for Telegram MTPROTO available. Here
are the most notable:

* [Official](https://github.com/TelegramMessenger/MTProxy)
* [Python](https://github.com/alexbers/mtprotoproxy)
* [Erlang](https://github.com/seriyps/mtproto_proxy)

Almost all of them follow the way how official proxy was build. This
includes support of multiple secrets, support of promoted channels etc.

mtg is an implementation in golang which is intended to be:

* **Lightweight**
  It has to consume as less resources as possible but not by losing
  maintainability.
* **Easily deployable**
  I strongly believe that Telegram proxies should follow the way of
  ShadowSocks: promoted channels is a strange way of doing business
  I suppose. I think the only viable way is to have a proxy with
  minimum configuration which should work everywhere.
* **Single secret**
  I think that multiple secrets solves no problems and just complexify
  software. I also believe that in case of throwout proxies, this feature
  is useless luxury.
* **Minimum docker image size**
  Official image is less than 2.5 megabytes. Literally.
* **No management WebUI**
  This is an implementation of simple lightweight proxy. I won't do that.

This proxy supports 2 modes of work: direct connection to Telegram and
promoted channel mode. If you do not need promoted channels, I would
recommend you to go with direct mode: this is way more robust.

To run proxy in direct mode, all you need to do is just provide a
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
$ make
```

If you want to build for another platform:

```console
$ make crosscompile
```

If you want to build Docker image (called `mtg`):

```console
$ make docker
```

# Docker image

Docker follows the same policy as the source code organization:

- `latest` mirrors the master branch
- `stable` mirrors the stable branch
- tags are for tagged releases

```console
$ docker pull nineseconds/mtg:latest
```

```console
$ docker pull nineseconds/mtg:stable
```

```console
$ docker pull nineseconds/mtg:0.10
```

# Configuration

Basically, to run this tool you need to configure as less as possible.

First, you need to generate a secret:

```console
$ openssl rand -hex 16
```

or

```console
$ head -c 512 /dev/urandom | md5sum | cut -f 1 -d ' '
```

## Secure mode

If you want to support new secure mode, please prepend `dd` to the
secret. For example, secret `cf18fa8ea0267057e2c61a5f7322a8e7` should
be `ddcf18fa8ea0267057e2c61a5f7322a8e7`. But pay attention that some
old clients won't support this mode. If this is not your case, I would
suggest to go with this mode.

Oneliners to generate such secrets:

```console
$ echo dd$(openssl rand -hex 16)
```

or

```console
$ echo dd$(head -c 512 /dev/urandom | md5sum | cut -f 1 -d ' ')
```


# How to run the tool

Now run the tool:

```console
$ mtg <secret>
```

How to run the tool with ADTag:

```console
$ mtg <secret> <adtag>
```

This tool will listen on port 3128 by default with the given secret.

# One-line runner

```console
$ docker run --name mtg --restart=unless-stopped -p 3128:3128 -p 3129:3129 -d nineseconds/mtg:stable $(openssl rand -hex 16)
```

or in secret mode:

```console
$ docker run --name mtg --restart=unless-stopped -p 3128:3128 -p 3129:3129 -d nineseconds/mtg:stable dd$(openssl rand -hex 16)
```

You will have this tool up and running on port 3128. Now curl
`localhost:3129` to get `tg://` links or do `docker logs mtg`. Also,
port 3129 will show you some statistics if you are interested in.

Also, you can use [run-mtg.sh](https://github.com/9seconds/mtg/blob/master/run-mtg.sh) script


# statsd integration

mtg provides an integration with statsd, you can enable it with command
line interface. To enable it, you have to provide IP address of statsd
service.

Out of the box, mtg supports 2 additional dialects: [InfluxDB](https://www.influxdata.com/blog/getting-started-with-sending-statsd-metrics-to-telegraf-influxdb/)
and [Datadog](https://docs.datadoghq.com/developers/dogstatsd/).

All metrics are gauges. Here is the list of metrics and their meaning:

| Metric name                     | Unit    | Description                                               |
|---------------------------------|---------|-----------------------------------------------------------|
| `connections.abridged.ipv4`     | number  | The number of active abridged IPv4 connections            |
| `connections.abridged.ipv6`     | number  | The number of active abridged IPv6 connections            |
| `connections.intermediate.ipv4` | number  | The number of active intermediate IPv4 connections        |
| `connections.intermediate.ipv6` | number  | The number of active intermediate IPv6 connections        |
| `connections.secure.ipv4`       | number  | The number of active secure intermediate IPv4 connections |
| `connections.secure.ipv6`       | number  | The number of active secure intermediate IPv6 connections |
| `crashes`                       | number  | An amount of crashes in client handlers                   |
| `traffic.ingress`               | bytes   | Ingress traffic from the start of application (incoming)  |
| `traffic.egress`                | bytes   | Egress traffic from the start of application (outgoing)   |
| `speed.ingress`                 | bytes/s | Ingress bandwidth of the latest second (incoming traffic) |
| `speed.egress`                  | bytes/s | Egress bandwidth of the latest second (outgoing traffic)  |

All metrics are prefixed with given prefix. Default prefix is `mtg`.
With such prefix metric name `traffic.ingress`, for example, would be
`mtg.traffic.ingress`.
