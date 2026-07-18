# Self-hosting JPRQ

This fork supports running JPRQ behind an existing reverse proxy without taking
over ports 80 and 443. Authentication can use either a private static token or
a self-hosted GitHub OAuth flow without the upstream membership service.

## Requirements

- Linux amd64 or arm64 server with systemd
- a base DNS record such as `jprq.example.com`
- a wildcard DNS record such as `*.jprq.example.com`
- Nginx or another reverse proxy
- Go 1.24+ for building

Both DNS records must resolve to the JPRQ server. The event port (4321 by
default) and dynamically allocated TCP tunnel ports must be reachable by JPRQ
clients. Do not expose the private HTTP backend port.

## Build

```sh
go test ./server/config ./server/events ./server/server ./server/tunnel
go build -trimpath -ldflags='-s -w' -o jprq-server ./server
go build -trimpath \
  -ldflags='-s -w -X main.remoteConfig=https://jprq.example.com/config.json -X main.publicScheme=http' \
  -o jprq ./cli
```

Use `main.publicScheme=https` only when a valid wildcard TLS certificate covers
every tunnel hostname.

## Install the server

```sh
sudo install -d -m 0755 /usr/local/lib/jprq /etc/jprq /var/www/jprq
sudo install -m 0755 jprq-server /usr/local/lib/jprq/jprq-server
sudo install -m 0600 deploy/jprq.env.example /etc/jprq/jprq.env
sudo install -m 0644 deploy/jprq.service /etc/systemd/system/jprq.service
```

Generate a token with `openssl rand -hex 32`, put it in
`/etc/jprq/jprq.env`, and replace the example domain. The recommended layout is:

- public event socket: `0.0.0.0:4321`
- private HTTP backend: `127.0.0.1:18080`
- TLS disabled in JPRQ when Nginx terminates TLS

### GitHub OAuth authentication

Create a GitHub OAuth App with this callback URL:

```text
https://jprq.example.com/oauth-callback
```

Remove `JPRQ_AUTH_TOKEN` and set `GITHUB_CLIENT_ID`,
`GITHUB_CLIENT_SECRET`, `GITHUB_REDIRECT_URI`, and `JPRQ_WEBSITE_ADDR` as shown
in `deploy/jprq.env.example`. Build and install the website service:

```sh
go build -trimpath -ldflags='-s -w' -o jprq-website ./website
sudo install -m 0755 jprq-website /usr/local/lib/jprq/jprq-website
sudo install -m 0644 deploy/jprq-website.service /etc/systemd/system/jprq-website.service
sudo systemctl enable --now jprq-website
```

Proxy `/auth` and `/oauth-callback` to the loopback website address. Users then
visit `https://jprq.example.com/auth`, authorize with GitHub, and copy the
generated `jprq auth ...` command. OAuth secrets must remain only in the mode
0600 environment file and must never be committed.

Install and adapt `deploy/nginx-http.conf.example`, run `nginx -t`, and only
then reload Nginx. Finally:

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now jprq
sudo systemctl status jprq
```

The client reads this file from `main.remoteConfig`:

```json
{"domain":"jprq.example.com","events":"jprq.example.com:4321"}
```

Install the custom client, authenticate once, and open a tunnel:

```sh
sudo install -m 0755 jprq /usr/local/bin/jprq
jprq auth YOUR_STATIC_TOKEN
jprq http 8000 -s demo
```

The HTTP example is then reachable at `http://demo.jprq.example.com`.

## HTTPS tunnels

Wildcard hostnames require a wildcard certificate. Let's Encrypt issues these
through DNS-01 validation, so configure the appropriate Certbot DNS plugin or
another ACME client for your DNS provider. Never place a broad DNS API key in a
world-readable file. After obtaining the certificate, either terminate wildcard
TLS in Nginx or enable JPRQ TLS with `JPRQ_TLS_CERT`, `JPRQ_TLS_KEY`, and a
non-conflicting HTTPS bind port.

## Security notes

- Keep `/etc/jprq/jprq.env` mode 0600.
- Use a random token of at least 32 bytes and rotate it if disclosed.
- Keep the HTTP backend bound to loopback.
- JPRQ intentionally opens random public ports for TCP tunnels. Restrict them
  at the network firewall if TCP tunneling is not needed.
- Test Nginx configuration before every reload.
- Run the systemd service without root; the included unit uses `DynamicUser`.
