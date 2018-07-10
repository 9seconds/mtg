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
* [JS](https://github.com/FreedomPrevails/JSMTProxy)

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

```console
$ docker pull nineseconds/mtg
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
$ docker run --name mtg --restart=unless-stopped -p 3128:3128 -p 3129:3129 -d nineseconds/mtg $(openssl rand -hex 16)
```

or in secret mode:

```console
$ docker run --name mtg --restart=unless-stopped -p 3128:3128 -p 3129:3129 -d nineseconds/mtg dd$(openssl rand -hex 16)
```

You will have this tool up and running on port 3128. Now curl
`localhost:3129` to get `tg://` links or do `docker logs mtg`. Also,
port 3129 will show you some statistics if you are interested in.

Also, you can use [run-mtg.sh](https://github.com/9seconds/mtg/blob/master/run-mtg.sh) script
