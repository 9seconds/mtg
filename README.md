# mtg

Highly-opinionated (ex-bullshit-free) MTPROTO proxy for
[Telegram](https://telegram.org/).

[![CI](https://github.com/9seconds/mtg/actions/workflows/ci.yaml/badge.svg?branch=master)](https://github.com/9seconds/mtg/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/9seconds/mtg/branch/master/graph/badge.svg?token=JfdDyGVpT4)](https://codecov.io/gh/9seconds/mtg)
[![Go Reference](https://pkg.go.dev/badge/github.com/9seconds/mtg.svg)](https://pkg.go.dev/github.com/9seconds/mtg/v2)

**If you use v1.0 or upgrade broke you proxy, please read the chapter
[Version 2](#version-2)**

## Rationale

There are several available proxies for Telegram MTPROTO available. Here
are the most notable:

* [Official](https://github.com/TelegramMessenger/MTProxy)
* [Python](https://github.com/alexbers/mtprotoproxy)
* [Erlang](https://github.com/seriyps/mtproto_proxy)

You can use any of these. They work great and all implementations have
feature parity now. This includes support of adtag, replay attack
protection, domain fronting, faketls, and so on. mtg has a similar
goal: to give a possibility to connect to Telegram in a restricted,
censored environment. But it does it slightly differently in details
that probably matter.

* **Resource-efficient**

  It has to be resource-efficient. It does not mean that you will see
  the smallest memory usage. It means that it will try to use allocated
  resources in zero-waste mode, reusing as much memory as possible and
  so on.

* **Easily deployable**

  I strongly believe that Telegram proxies should follow the way of
  [ShadowSocks](https://shadowsocks.org): promoted channels is a strange
  way of doing business I suppose. I think the only viable way is to
  have a proxy that can be restored anywhere easily.

* **A single secret**

  I think that multiple secrets solve no problems and just complex
  software. I also believe that in the case of throwout proxies, this
  the feature is a useless luxury.

* **No adtag support**

  Please read [Version 2](#version-2) chapter.

* **No management WebUI**

  This is an implementation of a simple lightweight proxy. I won't do that.

* **Proxy chaining**

  mtg has the support of [SOCKS5](https://en.wikipedia.org/wiki/SOCKS)
  proxies. So, in theory, you can run this proxy as a frontend
  and route traffic via [v2ray](https://www.v2ray.com/),
  [Gost](https://docs.ginuerzh.xyz/gost/),
  [Trojan](https://trojan-gfw.github.io/trojan/), or any other project
  you like.

* **Native blocklist support**

  Previously, this was delegated to the [FireHOL](https://firehol.org/)
  project or similar ones which track attacks and publish a list of
  potentially dangerous IPs. mtg has native support of such blocklists.

* **Can be used as a library**

  mtg v2 was redesigned in a way so it can be embedded into your
  software (written in Golang) with a minimum effort + you can replace
  some parts with those you want.

### Version 2

If you use version 1.x before, you are probably noticed some major
backward non-compatible details:

1. Configuration file
2. Removed support of adtag

For the configuration file, please check out the full example in this
repository. It has a lot of comments and most of the options are
optional. We do have only `secret` and `bind-to` sections mandatory.
Other sections in the example configuration file are filled with default
values.

Adtag support was removed completely. This was done to debloat mtg and
keep it simple and obvious. Hopefully, this goal is achieved and the
source code is clean and straightforward enough.

I always was quite skeptical about adtag. In my POV, a proxy as a fat
big connectivity point for hundreds of clients is an illusion. If you
work in a censored environment, the first thing that authority does is
IP blocking. For us, it means, those big proxies that can benefit from
having a pinned channel are going to be blocked in a minute.

Proxy has to be intimate. It has to be shared within a small group as
a family or maybe your college friends. It has to have a small number
of connections and never publicly announced its presence. It has to fly
under the radar. If the proxy is detected, you need to be able to give
a rebirth on a new IP address as soon as possible. I do no think that
having some special channel for such a use case makes any sense.

But other details like replay attack protection, domain fronting,
accurate FakeTLS implementation, IP blacklisting, and proxy
chaining matter here. If you work in censored perimeter like
[GFW](https://en.wikipedia.org/wiki/Great_Firewall)-protected
country, you probably want to have an MTPROTO proxy as
a frontend that transports traffic via cloaked tunnels
made by [Trojan](https://trojan-gfw.github.io/trojan/),
[Shadowsocks](https://shadowsocks.org), [v2ray](https://www.v2ray.com/),
or [Gost](https://docs.ginuerzh.xyz/gost/). That's why you have to have
the support of chaining as a first-class citizen.

Yes, this is possible and doable with optional adtag support. But the
truth is that the MTPROTO proxy for Telegram is just a thing that either
work as a normal client (direct mode) or doing some RPC calls in [TL
language](https://core.telegram.org/mtproto/TL) (adtag support). I
understand the intention of the developers and I understand that they
were under high pressure fighting with [RKN](https://rkn.gov.ru/) and
doing TON after that. Nothing is ideal. But for the proxy, it means that
source code is full of complex non-trivial code which is required only
to support a feature that we barely need.

So, to have a reasonable MTPROTO proxy, adtag support was removed. This
is a rare chance in my career where software v2 debloats a previous
version. It feels so good :)

### Version 1 and 2

I do continue to support both versions 1 and 2. But in a different mode.

Version 1 is now officially in maintenance mode. It means that I won't
make any new features or improvements there. You can consider a feature
freeze there. No bugs are going to be fixed there except for critical
ones. PRs are welcome though. The goal is to keep it working. It will
get some periodical updates like updates to the new Golang version of
dependencies version bump, but that's mostly it.

**If you want to have mtg with _adtag support_, please use version 1**.

Version 2 is going to have all my love, active support, bug fixing, etc.
It is under active development and maintenance.

This project has several main branches

1. [`master`](https://github.com/9seconds/mtg/tree/master) branch
   contains a bleeding edge. It may potentially have some features
   which will break your source code.
2. [`stable`](https://github.com/9seconds/mtg/tree/stable) branch contains
   dumps of a master branch when we consider it 'stable'. This is a
   branch you probably want to pick.
3. [`v2`](https://github.com/9seconds/mtg/tree/v2) has a development
   of the v2.x version. In theory, it is the same as `master` but this
   will change when we have v3.x.
4. [`v1`](https://github.com/9seconds/mtg/tree/v1) has a version 1.x.

## Getting started

### Download a tool

#### Download binaries

Binaries can be downloaded from the release page. Also, you can download
docker image.

For the current version, please download like

```console
docker pull nineseconds/mtg:2
```

For version 1:

```console
docker pull nineseconds/mtg:1
```

You may also check both [Docker
Hub](https://hub.docker.com/r/nineseconds/mtg/tags) and [Github
Registry](https://github.com/users/9seconds/packages/container/package/mtg).
Please do not choose `latest` or `stable` if you want to avoid
surprises. Always choose some version tag.

Also, if you have `go` installed, you can always download this tool with `go get`:

```console
go install github.com/9seconds/mtg/v2@latest
```

#### Build from sources

```console
git clone https://github.com/9seconds/mtg.git
cd mtg
make static
```

or for the docker image:

```console
make docker
```

### Generate secret

If you already have a secret in Base64 format or that, which starts with `ee`,
you can skip this chapter. Otherwise:

```console
$ mtg generate-secret google.com
7ibaERuTSGPH1RdztfYnN4tnb29nbGUuY29t
```

or

```console
$ mtg generate-secret --hex google.com
ee473ce5d4958eb5f968c87680a23854a0676f6f676c652e636f6d
```

equivalent commands with docker:

```console
$ docker run --rm nineseconds/mtg:2 generate-secret google.com
7ibaERuTSGPH1RdztfYnN4tnb29nbGUuY29t

$ docker run --rm nineseconds/mtg:2 generate-secret --hex google.com
ee473ce5d4958eb5f968c87680a23854a0676f6f676c652e636f6d
```

This secret is a keystone for a proxy and your password for a client.
You need to keep it secured.

We recommend choosing a hostname wisely. Here we have a _google.com_
but in reality, all providers can easily detect that this is not a
Google. Google has a list of networks it officially uses and your IP
address won't probably belong to it. It is a great idea to hide behind
some domain that has some relation to this IP address.

For example, you've bought a VPS from [Digital
Ocean](https://www.digitalocean.com/). Then it might be a good idea to
generate a secret for _digitalocean.com_ then.


### Simple run mode

mtg supports 2 modes: simple and normal. Simple mode allows starting
proxy with a small subset of configuration options you usually want to
modify. This is quite good for oneliners that you can copy-paste and do
not bother about external files whatsoever.

Let's take a look:

```console
Usage: mtg simple-run <bind-to> <secret>

Run proxy without config file.

Arguments:
  <bind-to>    A host:port to bind proxy to.
  <secret>     Proxy secret.

Flags:
  -h, --help                           Show context-sensitive help.
  -v, --version                        Print version.

  -d, --debug                          Run in debug mode.
  -c, --concurrency=8192               Max number of concurrent connection to proxy.
  -b, --tcp-buffer="4KB"               Size of TCP buffer to use.
  -i, --prefer-ip="prefer-ipv6"        IP preference. By default we prefer IPv6 with fallback to IPv4.
  -p, --domain-fronting-port=443       A port to access for domain fronting.
  -n, --doh-ip=9.9.9.9                 IP address of DNS-over-HTTP to use.
  -t, --timeout=10s                    Network timeout to use
  -a, --antireplay-cache-size="1MB"    A size of anti-replay cache to use.
```

So, if you want to startup a proxy with CLI only, you can do something like

```console
$ mtg simple-run -n 1.1.1.1 -t 30s -a 512kib 127.0.0.1:3128 7hBO-dCS4EBzenlKbdLFxyNnb29nbGUuY29t
```

The rest of the configuration will be taken from default values. But
a simple run is fine if you do not have any special requirements or
granular tuning. If you want it, please checkout the configuration
files.

### Prepare a configuration file

Please checkout an example configuration file. All options except of
`secret` and `bind-to` are optional. You can safely have this minimal
configuration file:

```toml
secret = "ee473ce5d4958eb5f968c87680a23854a0676f6f676c652e636f6d"
bind-to = "0.0.0.0:443"
```

This is enough to run the whole application. All other
options already have sensible defaults for the app at almost any scale.

Oh, the configuration is done in [TOML format](https://toml.io/en/).

### Run a proxy

Put a binary and a config into your webserver. Just for example,
a binary goes to `/usr/local/bin/mtg` and configuration to `/etc/mtg.toml`.

Now you can create a systemd unit:

```console
$ cat /etc/systemd/system/mtg.service
[Unit]
Description=mtg - MTProto proxy server
Documentation=https://github.com/9seconds/mtg
After=network.target

[Service]
ExecStart=/usr/local/bin/mtg run /etc/mtg.toml
Restart=always
RestartSec=3
DynamicUser=true
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
$ sudo systemctl daemon-reload
$ sudo systemctl enable mtg
$ sudo systemctl start mtg
```

or you can run a docker image

```console
docker run -d -v $PWD/config.toml:/config.toml -p 443:3128 --name mtg-proxy --restart=unless-stopped nineseconds/mtg:2
```

where _443_ is a host port (a port you want to connect to from a
client), and _3128_ is the one you have in your config in the `bind-to`
section.

### Access a proxy

Now you can generate some useful links:

```console
$ mtg access /etc/mtg.toml
{
  "ipv4": {
    "ip": "x.y.z.a",
    "port": 3128,
    "tg_url": "tg://proxy?...",
    "tg_qrcode": "https://api.qrserver.com/v1/create-qr-code?data...",
    "tme_url": "https://t.me/proxy?...",
    "tme_qrcode": "https://api.qrserver.com/v1/create-qr-code?data..."
  },
  "secret": {
    "hex": "...",
    "base64": "..."
  }
}
```

or if you are using docker:

```console
$ docker exec mtg-proxy /mtg access /config.toml
```

## Metrics

Out of the box, mtg works with
[statsd](https://github.com/statsd/statsd) and
[Prometheus](https://prometheus.io/). Please check configuration file
example to get how to set this integration up.

Here goes a list of metrics with their types but without a prefix.

| Name                        | Type    | Tags                             | Description                                                                                |
|-----------------------------|---------|----------------------------------|--------------------------------------------------------------------------------------------|
| client_connections          | gauge   | `ip_family`                      | Count of processing client connections.                                                    |
| telegram_connections        | gauge   | `telegram_ip`, `dc`              | Count of connections to Telegram servers.                                                  |
| domain_fronting_connections | gauge   | `ip_family`                      | Count of connections to fronting domain.                                                   |
| iplist_size                 | gauge   | `ip_list`                        | A size of either allowlist or blocklist in use.                                            |
| telegram_traffic            | counter | `telegram_ip`, `dc`, `direction` | Count of bytes, transmitted to/from Telegram.                                              |
| domain_fronting_traffic     | counter | `direction`                      | Count of bytes, transmitted to/from fronting domain.                                       |
| domain_fronting             | counter | –                                | Count of domain fronting events.                                                           |
| concurrency_limited         | counter | –                                | Count of events, when client connection was rejected due to concurrency limit.             |
| ip_blocklisted              | counter | `ip_list`                        | Count of events when client connection was rejected because IP was found in the blocklist. |
| replay_attacks              | counter | –                                | Count of detected replay attacks.                                                          |

Tag meaning:

| Name        | Values                     | Description                                   |
|-------------|----------------------------|-----------------------------------------------|
| ip_family   | `ipv4`, `ipv6`             | A version of the IP protocol.                 |
| dc          |                            | A number of the Telegram DC for a connection. |
| telegram_ip |                            | IP address of the Telegram server.            |
| direction   | `to_client`, `from_client` | A direction of the traffic flow.              |
| ip_list     | `allowlist`, `blocklist`   | A type of the IP list.                        |
