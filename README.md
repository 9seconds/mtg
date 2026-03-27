# mtg

Highly-opinionated (ex-bullshit-free) MTPROTO proxy for
[Telegram](https://telegram.org/).

[![CI](https://github.com/9seconds/mtg/actions/workflows/ci.yaml/badge.svg?branch=master)](https://github.com/9seconds/mtg/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/9seconds/mtg/branch/master/graph/badge.svg?token=JfdDyGVpT4)](https://codecov.io/gh/9seconds/mtg)
[![Go Reference](https://pkg.go.dev/badge/github.com/9seconds/mtg.svg)](https://pkg.go.dev/github.com/9seconds/mtg/v2)

**If you use v1.0 or upgrade broke you proxy, please read the chapter
[Version 2](#version-2)**

If you want to have a proxy that _supports adtag_ (possibility to promote a
channel with a special Telegram bot), I recommend to use
[telemt](https://github.com/telemt/telemt) project. v1 of mtg supports it
but I do not see any reasonable point of using it: adtag requires communication
via a fragile set of middle proxies, requires complex setup that must expose
a public IPs, has lower bandwidth and latency.

mtg idea is simple: minimal unbloated proxy that can handle a reasonable scale
~10-20k simultaneous connections, has no user management, but ticks all
checkboxes related to its main intent: provide a way to use Telegram.

## Getting Started

### Docker (Recommended)
```bash
docker run -d --name mtg --restart=unless-stopped -p 3128:3128 \
    n0vad3v/mtg run <SECRET_HERE>
```

### From Source
```bash
go install github.com/9seconds/mtg/v2@latest
mtg run <SECRET_HERE>
```

## Rationale

There are several available proxies for Telegram MTPROTO available. Here
are the most notable:

* [Official](https://github.com/TelegramMessenger/MTProxy)
* [Python](https://github.com/alexbers/mtprotoproxy)
* [Erlang](https://github.com/seriyps/mtproto_proxy)
* [Telemt (Rust)](https://github.com/telemt/telemt)

You can use any of these. They work great and all implementations have
feature parity now. This includes support of adtag, replay attack
protection, domain fronting, faketls, and so on. mtg has a similar
goal: to give a possibility to connect to Telegram in a restricted,
censored environment. But it does it slightly differently in details
that probably matter.

* **Domain fronting**

  For years mtg supports domain fronting. This technique means that it fallbacks
  to accessing a real website in case if request fails. It could fail by many
  reasons: anti-replay protection, accidental access to the webserver or
  stale request. Anyway, if mtg rejects this request, it does not break a
  connection. It connects to the websites and replicates everything that client
  has sent, and simply proxies it back as is. Users will see a response from
  the real website, _byte-to-byte identical_ to the response of the real netloc.

* **Doppelganger**

  mtg also is a doppelganger of the website it fronts. Sure, with domain fronting
  users will see replies of the real website in case if something will go wrong.
  But what about such cases when _everything is fine_?

  In that case mtg mimics TLS connection statistical characteristics as close as
  possible. Different application have different statistics of their patterns.
  Big CDN steadily pumping the data, small websites burst with short easily
  compressiable chunks of traffic