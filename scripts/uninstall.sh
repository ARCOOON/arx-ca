#!/usr/bin/env bash
#
# uninstall.sh — Stop the arx CA server, remove the systemd unit, back up data, and delete the install.
# Must be run as root on a Linux system with systemd.
#
set -e

SERVICE_NAME="arx-server"
APP_DIR="/opt/arx"
SYSTEM_USER="arx-ca"
UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Error: This script must be run as root." >&2
  exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo "Error: systemd (systemctl) is required but was not found." >&2
  exit 1
fi

echo "=== Uninstalling arx CA server ==="

if systemctl list-unit-files "${SERVICE_NAME}.service" >/dev/null 2>&1; then
  echo "Stopping ${SERVICE_NAME} service..."
  systemctl stop "${SERVICE_NAME}" 2>/dev/null || true

  echo "Disabling ${SERVICE_NAME} service..."
  systemctl disable "${SERVICE_NAME}" 2>/dev/null || true
else
  echo "Service ${SERVICE_NAME} is not registered; skipping stop/disable."
fi

if [[ -f "${UNIT_FILE}" ]]; then
  echo "Removing systemd unit ${UNIT_FILE}..."
  rm -f "${UNIT_FILE}"
  echo "Reloading systemd daemon..."
  systemctl daemon-reload
else
  echo "Systemd unit ${UNIT_FILE} not found; skipping removal."
fi

if [[ -d "${APP_DIR}" ]]; then
  TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
  BACKUP_DIR="/root/arx_backup_${TIMESTAMP}"
  echo "Backing up ${APP_DIR} to ${BACKUP_DIR}..."
  cp -a "${APP_DIR}" "${BACKUP_DIR}"
  echo "Removing application directory ${APP_DIR}..."
  rm -rf "${APP_DIR}"
else
  echo "Application directory ${APP_DIR} does not exist; skipping backup and removal."
fi

if id -u "${SYSTEM_USER}" >/dev/null 2>&1; then
  echo "Removing system user ${SYSTEM_USER}..."
  userdel "${SYSTEM_USER}" 2>/dev/null || userdel -f "${SYSTEM_USER}"
else
  echo "System user ${SYSTEM_USER} does not exist; skipping user removal."
fi

if getent group "${SYSTEM_USER}" >/dev/null 2>&1; then
  echo "Removing system group ${SYSTEM_USER}..."
  groupdel "${SYSTEM_USER}" 2>/dev/null || true
else
  echo "System group ${SYSTEM_USER} does not exist; skipping group removal."
fi

echo ""
echo "=== Uninstallation complete ==="
if [[ -n "${BACKUP_DIR:-}" && -d "${BACKUP_DIR}" ]]; then
  echo "Configuration and data were backed up to: ${BACKUP_DIR}"
fi
