# arx-agent configuration and renewal

The **arx-agent** binary is the data-plane renewal daemon. It reads **`agent.yaml` only** — it never loads `server.yaml`. Server settings belong to the `arx` control-plane binary.

## Configuration file location

| Context | Path |
| ------- | ---- |
| Default | `~/.arx-cert-service/agent.yaml` |
| Production install | `<install-dir>/agent.yaml` (for example `/opt/arx-agent/agent.yaml`) |
| Override | `arx-agent run --config /path/to/agent.yaml` |

Generate a starter file with examples for both renewal protocols:

```bash
arx-agent config init
arx-agent config init --config /opt/arx-agent/agent.yaml
arx-agent config init --force
```

## Top-level structure

```yaml
daemon:
  check_interval: 24h
  renew_threshold: 720h
  managed_certs:
    - ...
```

| Field | Default | Description |
| ----- | ------- | ----------- |
| `daemon.check_interval` | `24h` | How often the daemon evaluates managed certificates |
| `daemon.renew_threshold` | `720h` | Renew when remaining TTL is below this value |
| `daemon.managed_certs` | `[]` | List of certificate targets (each entry chooses a protocol) |

Environment overrides use the `ARX_AGENT_` prefix (for example `ARX_AGENT_DAEMON_CHECK_INTERVAL=12h`).

## Managed certificate entries

Every `managed_certs` item requires:

| Field | Required | Description |
| ----- | -------- | ----------- |
| `cert_path` | yes | Path to the certificate PEM file |
| `key_path` | yes | Path to the private key PEM file |
| `common_name` | yes | Primary DNS name for the certificate |
| `protocol` | no | `api` (default) or `acme` |
| `post_hook` | no | Shell command after successful renewal (for example `systemctl reload nginx`) |

### Native API renewal (`protocol: api`)

Uses the Arx REST API (`POST /api/v1/certificates/auto`) with credentials from `arx login` (`~/.arx/config.json`).

| Field | Required | Description |
| ----- | -------- | ----------- |
| `template` | no | Certificate template / profile name (`template_id` in the API) |

Example:

```yaml
managed_certs:
  - protocol: api
    cert_path: /etc/nginx/ssl/app.pem
    key_path: /etc/nginx/ssl/app-key.pem
    template: web-server
    common_name: app.internal.example
    post_hook: systemctl reload nginx
```

**Prerequisites:** run `arx login --url https://ca.example.com` on the agent host (or copy `~/.arx/config.json`).

### ACME client renewal (`protocol: acme`)

Acts as an RFC 8555 ACME client (similar to Certbot) against any ACME directory — including the Arx CA at `/acme/directory` or a public CA such as Let's Encrypt.

| Field | Required | Description |
| ----- | -------- | ----------- |
| `acme_directory_url` | yes | ACME directory URL (for example `https://ca.example.com/acme/directory`) |
| `acme_email` | yes | Contact address for ACME account registration |
| `challenge_type` | no | Default `http-01` (only `http-01` is implemented) |
| `webroot` | one of | Document root for HTTP-01 token files (`.well-known/acme-challenge/`) |
| `challenge_listen_port` | one of | Standalone HTTP listener port when `webroot` is empty (default `80`) |

`webroot` and `challenge_listen_port` are mutually exclusive.

Example (webroot — typical behind nginx):

```yaml
managed_certs:
  - protocol: acme
    cert_path: /etc/nginx/ssl/public.pem
    key_path: /etc/nginx/ssl/public-key.pem
    common_name: www.example.com
    acme_directory_url: https://ca.example.com/acme/directory
    acme_email: ops@example.com
    challenge_type: http-01
    webroot: /var/www/html
    post_hook: systemctl reload nginx
```

Example (temporary listener on port 8080):

```yaml
managed_certs:
  - protocol: acme
    cert_path: /etc/ssl/acme.pem
    key_path: /etc/ssl/acme-key.pem
    common_name: edge.example.com
    acme_directory_url: https://acme-v02.api.letsencrypt.org/directory
    acme_email: ops@example.com
    challenge_listen_port: 8080
```

**ACME account state** is stored under `~/.arx-cert-service/acme/` (account key and registration KID), separate from `agent.yaml`.

**No `arx login` required** for ACME-managed entries.

## Running the daemon

```bash
arx-agent run
arx-agent run --config /opt/arx-agent/agent.yaml
```

On Linux, `sudo arx-agent service install` copies the binary, bootstraps a minimal `agent.yaml` if missing, and starts `arx-agent.service`.

## Mixed deployments

A single `agent.yaml` may list both API and ACME entries. The daemon routes each `managed_certs` item to the correct renewer based on `protocol`.

```yaml
daemon:
  check_interval: 12h
  renew_threshold: 168h
  managed_certs:
    - protocol: api
      cert_path: /etc/pki/internal.pem
      key_path: /etc/pki/internal-key.pem
      template: internal-server
      common_name: svc.corp.local
    - protocol: acme
      cert_path: /etc/pki/public.pem
      key_path: /etc/pki/public-key.pem
      common_name: www.example.com
      acme_directory_url: https://ca.example.com/acme/directory
      acme_email: ops@example.com
      webroot: /var/www/html
```

## Related documentation

- [cli_reference.md](cli_reference.md) — full `arx-agent` command reference
- [acme.md](acme.md) — ACME server directory and challenge types on the CA
- [architecture.md](architecture.md) — control plane vs data plane split
