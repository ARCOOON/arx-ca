#!/usr/bin/env bash
# uninstall.sh — Remove arx binary, WebUI assets, and symlink for the selected scope.
# Usage: uninstall.sh [--user|--system]   (default: --user)

set -euo pipefail

SCOPE="user"

usage() {
  cat <<EOF
Usage: $(basename "$0") [--user|--system]

  --user    Remove user-scoped installation (default)
            Directory: \$HOME/.arx
            Symlink:   \$HOME/.local/bin/arx

  --system  Remove system-wide installation (requires root)
            Directory: /opt/arx
            Symlink:   /usr/local/bin/arx

If server.yaml or .pki/ remain in the install directory, the directory is retained.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --user)
      SCOPE="user"
      shift
      ;;
    --system)
      SCOPE="system"
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "Error: unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ "${SCOPE}" == "system" ]]; then
  if [[ $(id -u) -ne 0 ]]; then
    echo "Error: --system requires root privileges. Run with sudo." >&2
    exit 1
  fi
  INSTALL_DIR="/opt/arx"
  SYMLINK="/usr/local/bin/arx"
else
  INSTALL_DIR="${HOME}/.arx"
  SYMLINK="${HOME}/.local/bin/arx"
fi

remove_if_exists() {
  local path="$1"
  if [[ -e "${path}" || -L "${path}" ]]; then
    rm -rf "${path}"
    echo "Removed ${path}"
  fi
}

main() {
  echo "Uninstalling arx (${SCOPE} scope)"
  echo "  Install directory: ${INSTALL_DIR}"

  if [[ -L "${SYMLINK}" ]] || [[ -e "${SYMLINK}" ]]; then
    rm -f "${SYMLINK}"
    echo "Removed symlink ${SYMLINK}"
  else
    echo "Symlink not found: ${SYMLINK} (skipped)"
  fi

  remove_if_exists "${INSTALL_DIR}/arx"
  remove_if_exists "${INSTALL_DIR}/ui"

  local preserved=false
  if [[ -f "${INSTALL_DIR}/server.yaml" ]]; then
    preserved=true
  fi
  if [[ -d "${INSTALL_DIR}/.pki" ]]; then
    preserved=true
  fi

  if [[ "${preserved}" == true ]]; then
    echo ""
    echo "Preserved data in ${INSTALL_DIR}:"
    [[ -f "${INSTALL_DIR}/server.yaml" ]] && echo "  - server.yaml"
    [[ -d "${INSTALL_DIR}/.pki" ]] && echo "  - .pki/"
    echo "Install directory retained: ${INSTALL_DIR}"
  elif [[ -d "${INSTALL_DIR}" ]]; then
    if rmdir "${INSTALL_DIR}" 2>/dev/null; then
      echo "Removed empty install directory ${INSTALL_DIR}"
    else
      echo "Install directory ${INSTALL_DIR} is not empty; retained remaining files."
    fi
  else
    echo "Install directory not found: ${INSTALL_DIR} (nothing to remove)"
  fi

  echo ""
  echo "Uninstall complete."
}

main
