# SNI-routing deployment for mtg

A turnkey `docker compose` setup that puts an SNI-aware TCP router
(HAProxy) in front of mtg **and** a real web server (Caddy with
automatic HTTPS).

## Why

Modern DPI systems actively probe suspected proxies.  If the server
closes the connection or returns something unexpected, the IP gets
flagged.  With this setup:

- **Telegram clients** connect to port 443, HAProxy sees the configured
  SNI and routes them to mtg (FakeTLS).
- **Everything else** (browsers, DPI probes, scanners) is routed to
  Caddy, which responds with a real Let's Encrypt certificate and serves
  genuine web content.

Because your domain's DNS points to this server, the SNI/IP match is
natural and passive DPI has nothing to flag.

## Quick start

```bash
# 1. Point your domain's DNS A/AAAA record to this server's IP.

# 2. Generate an mtg secret:
docker run --rm nineseconds/mtg:2 generate-secret --hex YOUR_DOMAIN

# 3. Edit the config files:
#    - mtg-config.toml  →  paste the secret
#    - haproxy.cfg       →  replace "example.com" in the SNI ACL
#    - .env or export    →  DOMAIN=your.domain

# 4. (Optional) put your site content into www/

# 5. Start:
docker compose up -d

# 6. Verify:
#    - Open https://YOUR_DOMAIN in a browser → you should see the web page
#    - Configure Telegram with the proxy link from:
docker compose exec mtg mtg access /config/config.toml
```

## Real client IPs (PROXY protocol)

HAProxy forwards TCP connections to mtg and Caddy with a PROXY protocol
v2 header so both backends see the real client IP instead of HAProxy's
container address.  The three pieces must stay in sync:

- `haproxy.cfg` — `send-proxy-v2` on the `mtg` and `web` backend `server` lines
- `mtg-config.toml` — `proxy-protocol-listener = true`
- `Caddyfile` — `listener_wrappers { proxy_protocol { ... } tls }` on `:8443`

If you disable one, disable all three, otherwise the backend will fail
to parse the connection.

## ACME (Let's Encrypt) notes

HAProxy passes `/.well-known/acme-challenge/` requests on `:80` to
Caddy so that HTTP-01 validation works out of the box.  Make sure your
domain's DNS A/AAAA record points to this server before starting.

## Architecture

```
              ┌──────────────────┐
 :443  ──────>│    HAProxy       │
              │  (TCP, SNI peek) │
              └──┬───────────┬───┘
    SNI match    │           │  default
                 v           v
           ┌─────────┐  ┌─────────┐
           │   mtg   │  │  Caddy  │
           │ :3128   │  │ :8443   │
           │ FakeTLS │  │ real TLS│
           └─────────┘  └─────────┘
```

## Files

| File | Purpose |
|---|---|
| `docker-compose.yml` | Service definitions |
| `haproxy.cfg` | SNI routing rules — **edit the domain** |
| `mtg-config.toml` | mtg proxy config — **paste your secret** |
| `Caddyfile` | Web server config (auto-HTTPS) |
| `www/` | Static site content served by Caddy |
