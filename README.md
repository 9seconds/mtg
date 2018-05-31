# mtg

[![Build Status](https://travis-ci.org/9seconds/mtg.svg?branch=master)](https://travis-ci.org/9seconds/mtg)
[![Docker Build Status](https://img.shields.io/docker/build/nineseconds/mtg.svg)](https://hub.docker.com/r/nineseconds/mtg/)

Bullshit-free MTPROTO proxy for Telegram

How to run:

```console
$ docker pull nineseconds/mtg
$ docker run --name mtg --restart=unless-stopped -p 3128:3128 -p 3129:3129 nineseconds/mtg aaabbbccc
```

Reasonable README with rationale will come a bit later, sorry.
