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

# 3. Configure:
#    - .env (or export)  →  DOMAIN=your.domain   # used by HAProxy + Caddy
#    - mtg-config.toml   →  paste the secret

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
container address.  Caddy also receives PROXY v2 from mtg on the
fronting path (see "Fronting loop" below), so all four pieces below
must stay in sync:

- `haproxy.cfg` — `send-proxy-v2` on the `mtg` and `web` backend `server` lines
- `mtg-config.toml` — `proxy-protocol-listener = true` (HAProxy → mtg)
- `mtg-config.toml` — `[domain-fronting].proxy-protocol = true` (mtg → Caddy on fronting)
- `Caddyfile` — `listener_wrappers { proxy_protocol { ... } tls }` on `:8443`

If you disable one, disable all four, otherwise the backend will fail
to parse the connection.

## Why host networking for HAProxy

HAProxy runs in the host network namespace (`network_mode: host` in
`docker-compose.yml`) so it sees the real client source IP on every
inbound connection.  With the default bridge networking + published
ports the source IP is rewritten to the bridge gateway — by Docker's
userland proxy (`docker-proxy`), by rootless Podman's `slirp4netns`
or `pasta`, or by NAT on the Docker host — and the PROXY v2 header
HAProxy then sends to mtg and Caddy carries that useless address.
Host networking lifts HAProxy out of the rewrite path; mtg and Caddy
stay on the compose bridge and HAProxy dials them via host loopback
(`127.0.0.1`).

Trade-off: HAProxy occupies the host's `:443` and `:80` directly, so
nothing else on the host may listen on those ports.  For a dedicated
mtg/SNI-router host that is the intended layout.

Rootless Podman users binding the privileged ports `:80`/`:443` need
the host-side sysctl once (rootful Docker handles this implicitly):

```bash
sudo sysctl -w net.ipv4.ip_unprivileged_port_start=80
```

## Fronting loop (why `[domain-fronting]` is set explicitly)

When mtg sees TLS that isn't valid Telegram (a probe or a browser
hitting the domain on `:443`), it forwards that connection to a real
web server — "domain fronting".  By default mtg uses the secret's
hostname as the fronting target and resolves it via DNS, which in
this setup points back to this server: the fronting dial lands on
HAProxy, SNI matches the secret, HAProxy routes the connection back
to mtg → loop.

The trigger is DNS, not name equality: any time the secret's hostname
resolves to this host, the loop reproduces.  In an SNI-router
deployment the secret's hostname has to point here for clients to
reach mtg in the first place, so the loop is the default state unless
mtg is steered away from HAProxy.

`mtg-config.toml` therefore pins the fronting target to the Caddy
container directly:

```toml
[domain-fronting]
host = "web"
port = 8443
proxy-protocol = true
```

`host = "web"` resolves through compose-network DNS to the `web`
service (Caddy), bypassing HAProxy.  `proxy-protocol = true` matches
Caddy's `:8443` listener wrapper so the real client IP still
propagates to Caddy's logs.

Requires mtg ≥ 2.4 — hostname acceptance for the fronting target was
added in #480.

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
| `haproxy.cfg` | SNI routing rules (reads `$DOMAIN` from the environment) |
| `mtg-config.toml` | mtg proxy config — **paste your secret** |
| `Caddyfile` | Web server config (auto-HTTPS) |
| `www/` | Static site content served by Caddy |
