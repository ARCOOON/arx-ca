#!/usr/bin/env bash
# install.sh — Install arx-ca and WebUI assets from the latest GitHub release.
# Usage: install.sh [--user|--system]   (default: --user)

set -euo pipefail

readonly REPO="ARCOOON/arx-ca"
readonly GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"

SCOPE="user"

usage() {
	cat <<EOF
Usage: $(basename "$0") [--user|--system]

  --user    Install for the current user (default)
            Directory: \$HOME/.arx-ca
            Symlink:   \$HOME/.local/bin/arx-ca

  --system  Install system-wide (requires root)
            Directory: /opt/arx-ca
            Symlink:   /usr/local/bin/arx-ca

Existing server.toml and .pki/ in the install directory are preserved on upgrade.
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
	INSTALL_DIR="/opt/arx-ca"
	SYMLINK="/usr/local/bin/arx-ca"
else
	INSTALL_DIR="${HOME}/.arx-ca"
	SYMLINK="${HOME}/.local/bin/arx-ca"
fi

detect_platform() {
	local os arch

	case "$(uname -s)" in
	Linux) os="linux" ;;
	Darwin) os="darwin" ;;
	*)
		echo "Error: unsupported operating system: $(uname -s)" >&2
		exit 1
		;;
	esac

	case "$(uname -m)" in
	x86_64 | amd64) arch="amd64" ;;
	aarch64 | arm64) arch="arm64" ;;
	*)
		echo "Error: unsupported architecture: $(uname -m)" >&2
		exit 1
		;;
	esac

	BINARY_NAME="arx-ca-${os}-${arch}"
}

fetch_latest_tag() {
	local response tag

	if ! response=$(curl -fsSL "${GITHUB_API}"); then
		echo "Error: failed to fetch latest release from GitHub API." >&2
		exit 1
	fi

	tag=$(printf '%s\n' "${response}" | sed -n 's/^[[:space:]]*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)

	if [[ -z "${tag}" ]]; then
		echo "Error: could not parse latest release tag from GitHub API response." >&2
		exit 1
	fi

	LATEST_TAG="${tag}"
}

preserve_user_data() {
	if [[ -f "${INSTALL_DIR}/server.toml" ]]; then
		echo "Preserving existing server.toml"
	fi
	if [[ -d "${INSTALL_DIR}/.pki" ]]; then
		echo "Preserving existing .pki/ directory"
	fi
}

install_webui() {
	local ui_dir="${INSTALL_DIR}/ui"

	mkdir -p "${ui_dir}"

	if [[ -d "${ui_dir}" ]] && [[ -n "$(ls -A "${ui_dir}" 2>/dev/null || true)" ]]; then
		find "${ui_dir}" -mindepth 1 -maxdepth 1 -exec rm -rf {} +
	fi

	tar -xzf "${TMPDIR}/webui-dist.tar.gz" -C "${ui_dir}"
}

ensure_user_path() {
	local local_bin="${HOME}/.local/bin"

	mkdir -p "${local_bin}"

	case ":${PATH}:" in
	*":${local_bin}:"*) ;;
	*)
		echo "Warning: ${local_bin} is not in your PATH."
		echo "Add the following to your shell profile (e.g. ~/.bashrc or ~/.zshrc):"
		echo '  export PATH="$HOME/.local/bin:$PATH"'
		;;
	esac
}

main() {
	detect_platform
	fetch_latest_tag

	echo "Installing arx-ca ${LATEST_TAG} (${SCOPE} scope)"
	echo "  Install directory: ${INSTALL_DIR}"
	echo "  Binary asset:      ${BINARY_NAME}"

	TMPDIR=$(mktemp -d)
	trap 'rm -rf "${TMPDIR}"' EXIT

	BASE_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}"

	echo "Downloading ${BINARY_NAME}..."
	curl -fsSL -o "${TMPDIR}/${BINARY_NAME}" "${BASE_URL}/${BINARY_NAME}"

	echo "Downloading webui-dist.tar.gz..."
	curl -fsSL -o "${TMPDIR}/webui-dist.tar.gz" "${BASE_URL}/webui-dist.tar.gz"

	mkdir -p "${INSTALL_DIR}"
	preserve_user_data

	echo "Installing binary to ${INSTALL_DIR}/arx-ca..."
	install -m 755 "${TMPDIR}/${BINARY_NAME}" "${INSTALL_DIR}/arx-ca"

	echo "Extracting WebUI assets to ${INSTALL_DIR}/ui/..."
	install_webui

	if [[ "${SCOPE}" == "user" ]]; then
		ensure_user_path
	fi

	echo "Creating symlink ${SYMLINK} -> ${INSTALL_DIR}/arx-ca..."
	ln -sf "${INSTALL_DIR}/arx-ca" "${SYMLINK}"

	echo ""
	echo "Installation complete."
	echo "  Version:  ${LATEST_TAG}"
	echo "  Binary:   ${INSTALL_DIR}/arx-ca"
	echo "  WebUI:    ${INSTALL_DIR}/ui/"
	echo "  Command:  arx-ca (via ${SYMLINK})"
	echo ""
	echo "Next steps:"

	if [[ "${SCOPE}" == "user" ]]; then
		echo "  arx-ca server config init --config ${INSTALL_DIR}/server.toml"
		echo "  arx-ca server start --config ${INSTALL_DIR}/server.toml"
		echo ""
		echo "Note: If running on a headless server, enable linger to allow the user service to start at boot:"
		echo "sudo loginctl enable-linger $USER"
	else
		echo "  arx-ca server config init --config ${INSTALL_DIR}/server.toml"
		echo "  arx-ca server start --config ${INSTALL_DIR}/server.toml"
	fi

	echo ""
}

main
