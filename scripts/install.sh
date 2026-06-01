#!/usr/bin/env bash
#
# install.sh — Install the arx CA server binary, configuration, and systemd unit.
# Must be run as root on a Linux system with systemd.
#
set -e

APP_NAME="arx"
SERVICE_NAME="arx-server"
APP_DIR="/opt/arx"
SYSTEM_USER="arx-ca"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SOURCE_BINARY="${PROJECT_ROOT}/bin/${APP_NAME}"
DEST_BINARY="${APP_DIR}/${APP_NAME}"
CONFIG_FILE="${APP_DIR}/server.yaml"
UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Error: This script must be run as root." >&2
  exit 1
fi

if [[ ! -f "${SOURCE_BINARY}" ]]; then
  echo "Error: Binary not found at ${SOURCE_BINARY}." >&2
  echo "Build it first from the project root: make build" >&2
  exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo "Error: systemd (systemctl) is required but was not found." >&2
  exit 1
fi

echo "=== Installing ${APP_NAME} CA server ==="

echo "Ensuring system group ${SYSTEM_USER} exists..."
if ! getent group "${SYSTEM_USER}" >/dev/null 2>&1; then
  groupadd --system "${SYSTEM_USER}"
  echo "Created system group ${SYSTEM_USER}."
else
  echo "System group ${SYSTEM_USER} already exists."
fi

echo "Ensuring system user ${SYSTEM_USER} exists..."
if ! id -u "${SYSTEM_USER}" >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin -g "${SYSTEM_USER}" "${SYSTEM_USER}"
  echo "Created system user ${SYSTEM_USER}."
else
  echo "System user ${SYSTEM_USER} already exists."
fi

echo "Creating application directory ${APP_DIR}..."
mkdir -p "${APP_DIR}"

echo "Installing binary to ${DEST_BINARY}..."
install -m 700 -o root -g root "${SOURCE_BINARY}" "${DEST_BINARY}"

if [[ ! -f "${CONFIG_FILE}" ]]; then
  echo "No server configuration found; generating default ${CONFIG_FILE}..."
  (cd "${APP_DIR}" && "${DEST_BINARY}" server config init)
else
  echo "Configuration file ${CONFIG_FILE} already exists; skipping config init."
fi

echo "Applying ownership and permissions..."
chown -R "${SYSTEM_USER}:${SYSTEM_USER}" "${APP_DIR}"
chmod 700 "${APP_DIR}"
chmod 700 "${DEST_BINARY}"
if [[ -f "${CONFIG_FILE}" ]]; then
  chmod 600 "${CONFIG_FILE}"
fi

echo "Writing systemd unit ${UNIT_FILE}..."
cat >"${UNIT_FILE}" <<EOF
[Unit]
Description=ARX Certificate Authority Server
Documentation=https://github.com/your-org/arx-ca
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SYSTEM_USER}
Group=${SYSTEM_USER}
WorkingDirectory=${APP_DIR}
ExecStart=${DEST_BINARY} server start --config ${CONFIG_FILE}
Restart=on-failure
RestartSec=5

# Hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectSystem=full

[Install]
WantedBy=multi-user.target
EOF
chmod 644 "${UNIT_FILE}"

echo "Reloading systemd daemon..."
systemctl daemon-reload

echo "Enabling and starting ${SERVICE_NAME}..."
systemctl enable "${SERVICE_NAME}"
systemctl start "${SERVICE_NAME}"

echo ""
echo "=== Installation complete ==="
echo "Service:  ${SERVICE_NAME}"
echo "Binary:   ${DEST_BINARY}"
echo "Config:   ${CONFIG_FILE}"
echo ""
echo "Check status:  systemctl status ${SERVICE_NAME}"
echo "View logs:     journalctl -u ${SERVICE_NAME} -f"
echo ""
echo "Edit ${CONFIG_FILE} (JWT secret, bootstrap password hash) before production use."
