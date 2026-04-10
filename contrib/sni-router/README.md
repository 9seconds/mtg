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

## ACME (Let's Encrypt) notes

Caddy needs to answer the ACME HTTP-01 challenge on port 80.  The
default `haproxy.cfg` redirects all `:80` traffic to HTTPS.  If Caddy
cannot obtain a certificate, either:

1. Temporarily stop HAProxy, let Caddy bind `:80` directly for the
   initial certificate, then start the full stack; or
2. Use DNS-01 validation in the Caddyfile (requires a DNS provider
   plugin); or
3. Add an HAProxy ACL that passes `/.well-known/acme-challenge/`
   requests to the Caddy backend instead of redirecting.

## Architecture

```
              ┌──────────────────┐
 :443  ──────>│    HAProxy       │
              │  (TCP, SNI peek) │
              └──┬───────────┬───┘
    SNI match    │           │  default
                 v           v
           ┌─────────┐  ┌─────────┐
           │   mtg    │  │  Caddy  │
           │ :3128    │  │ :8443   │
           │ FakeTLS  │  │ real TLS│
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
