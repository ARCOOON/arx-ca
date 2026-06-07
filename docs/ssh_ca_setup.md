# SSH CA Trust Setup

This guide explains how to configure Linux hosts and clients to trust certificates issued by the arx SSH Certificate Authority (step-ca SSH engine).

## 1. Retrieve the CA public keys

Download the SSH CA root public keys from the API or WebUI:

```bash
curl -sS https://ca.example.com/api/v1/ssh/roots | jq .
```

Save the **User CA** public key for server `sshd` trust and the **Host CA** public key for client `known_hosts` trust.

From the WebUI (`/ssh`), use **Download .pub** under **SSH CA Roots**.

## 2. Trust user certificates on a server (`sshd`)

User certificates authenticate clients. Install the **User CA** public key on every SSH server that should accept CA-signed logins.

1. Copy the User CA public key to the server:

```bash
sudo install -m 644 -o root -g root user-ca.pub /etc/ssh/ca.pub
```

2. Add or update `sshd_config`:

```ssh-config
# /etc/ssh/sshd_config
TrustedUserCAKeys /etc/ssh/ca.pub
```

3. Validate and reload:

```bash
sudo sshd -t && sudo systemctl reload sshd
```

Clients present a `-cert.pub` file (generated via `POST /api/v1/ssh/generate/user`) alongside their private key:

```bash
ssh -i ~/.ssh/id_ed25519 -i ~/.ssh/id_ed25519-cert.pub user@server.example.com
```

Or configure `IdentityFile` entries in `~/.ssh/config`.

## 3. Trust host certificates on a client

Host certificates let clients verify server identity without per-host `known_hosts` entries.

1. Save the **Host CA** public key locally:

```bash
mkdir -p ~/.ssh
curl -sS https://ca.example.com/api/v1/ssh/roots \
  | jq -r '.data.host_keys[0].public_key' >> ~/.ssh/known_hosts
```

Prefix the line with `@cert-authority` so OpenSSH treats it as a CA key:

```
@cert-authority * ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...
```

2. On each server, generate a host certificate and configure `sshd`:

```bash
# Sign the server host public key via the API or WebUI (Host Certificates tab)
# Install the signed certificate:
sudo install -m 644 -o root -g root ssh-host-cert.pub /etc/ssh/ssh_host_ed25519_key-cert.pub
```

Add to `sshd_config` (path must match your host key type):

```ssh-config
HostCertificate /etc/ssh/ssh_host_ed25519_key-cert.pub
```

Reload `sshd` after validation.

## 4. Extract the CA public key from PKI files

If the API is unavailable, read the on-disk SSH CA public keys from the step-ca PKI directory (default `.pki/`):

| Key | Typical path |
| --- | ------------ |
| User CA | `.pki/secrets/ssh_user_ca_key.pub` |
| Host CA | `.pki/secrets/ssh_host_ca_key.pub` |

Display a key:

```bash
cat .pki/secrets/ssh_user_ca_key.pub
```

The output is a single line in OpenSSH `authorized_keys` format (`ssh-ed25519 AAAA… comment`).

## 5. Operational notes

- User certificate default TTL is **4h**; host certificate default TTL is **8760h** (one year).
- User certificates include `permit-pty` and `permit-port-forwarding` extensions for interactive sessions.
- Audit events `SSH_USER_CERT_ISSUE` and `SSH_HOST_CERT_ISSUE` are recorded for each generation request.
- See [docs/api_reference.md](api_reference.md) for full REST schemas.
