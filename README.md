# mtg

Bullshit-free MTPROTO proxy for Telegram

[![Build Status](https://travis-ci.org/9seconds/mtg.svg?branch=master)](https://travis-ci.org/9seconds/mtg)
[![Docker Build Status](https://img.shields.io/docker/build/nineseconds/mtg.svg)](https://hub.docker.com/r/nineseconds/mtg/)


# Rationale

Telegram supports proxies and proxies act as a shield for censorship
and blocking actions of different goverments. At the moment of writing,
Telegram supports 2 types of proxies:

1. SOCKS5
2. MTPROTO

SOCKS5 proxy is general SOCKS proxy as defined in
[RFC1928](https://www.ietf.org/rfc/rfc1928.txt). The problem is that
by default SOCKS5 proxy has an access to the whole internet so a lot
of people tend to hide them "just for a case". It is possible to setup
SOCKS5 proxy so it is able to access just some IPs/CIDRs but, you know,
yeah.

MTPROTO proxy is a native Telegram proxy. It has several advantages:

1. Traffic is obfuscated by AES-CTR;
2. It allows connections only to Telegram services;
3. It gives proxy maintainer an ability to promote its channel.

But in reality, MTPROTO have 2 advantages (from my biased view):

1. Obfuscation
2. Simplify connection chain.

Here is how it looks like to work with SOCKS5 proxy:

```
Client -> SOCKS -> MTPROTO -> Telegram
```

SOCKS5 connects to IPs of Telegram proxies. AFAIK this is because
Telegram wants us to avoid censorship and regulations.

What MTPROTO proxies do:

```
Client -> MTPROTO -> Telegram
```

And promoted channels. I do not tend to use them because mtg was created
for slightly other way of using it but yeah. People want moneys.

There are a number of unofficial proxies and one
[OFFICIAL](https://github.com/TelegramMessenger/MTProxy), so why bother?

<start-biased-rant>

I'm a big fan of [ShadowSocks](http://www.shadowsocks.org/en/index.html)
project and I like how people use it. The majority of SS proxies are
disposable ones which are blocked/unblocked frequently. There are some
public lists of them in Internet so if one proxy has stopped to work,
you throw it out and use another one.

Some SS proxies are long-living. This is because they are not public and
intended to be used only by limited number of people. And single secret
is fine there.

What I do not get about official and some unofficial implementation is
why they decided to support multiple secrets? I mean, WTF with all of
you?

1. MTPROTO obfuscation (called obfuscated2) does not allow to verify
   client easily. You need to decrypt the frame for every secret. So, you
   need a number of workers which will constantly try to crack initial
   handshake frames with a list of secrets. That does not scale and will
   never be.

2. Why do you need a multiple secrets? Which task are you trying to
   solve with them? Valid secret means only 1 thing: access to Telegram. A
   binary thing. Absurd and rudimentarty access control.

Okay, you want to revoke an access, thats fine. Will you ssh to the
machine and restart the container? Do you want to have API for that? Web
UI? Maybe store secrets in database and collect statisitcs per each?

With all respect, this is idiotic thing. Guysngals, this is a proxy.
Gateway to Telegram. This is not a webservice, or SASS or name that
shit. This is disposable stuff. Blocked? Fine, go to the next one. Just
look at ShadowSocks. There is multiple user implementation available,
with control you want. Does anyone gives a flying fuck about it?

> Those Who Do Not Learn History Are Doomed To Repeat It
- George Santayana

What I want to have?

1. Minimal tool for me and my friends (which are not all my FB friends but
   a limited number of close friends).
2. Minimum viable configuration.
3. Single artifact runnable on every platform (not always Docker, some
   environments may have no Docker)
4. Smallest Docker image
5. Lightweight
6. Have as less management as possible.

</end-biased-rant>

So, please do not ask for:

1. Multiple users/secrets
2. Web UI
3. Detailed statistics/histograms etc.


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
$ head -c 512 | sha1sum | cut -f 1 -d ' '
```

Now run the tool:

```console
$ mtg <secret>
```

This tool will listen on port 3128 by default with the given secret.

# One-line runner

```
$ docker run --name mtg --restart=unless-stopped -p 444:3128 -p 3129:3129 -d nineseconds/mtg -a 444 $(openssl rand -hex 16)
```

You will have this tool up and running on port 444. Now curl
`localhost:3129` to get `tg://` links or do `docker logs mtg`. Also,
port 3129 will show you some statistics if you are interested in.
