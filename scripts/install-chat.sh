#!/usr/bin/env bash
set -euo pipefail

REPO="flowline-io/flowbot"
CLI_ASSET="flowbot-chat"
BINARY_NAME="flowbot-chat"
INSTALL_DIR="/usr/local/bin"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

VERSION="latest"
NO_VERIFY=false
TMPDIR=""

info()  { echo -e "  ${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "  ${RED}[ERROR]${NC} $*" >&2; }

cleanup() {
	if [[ -n "${TMPDIR}" && -d "${TMPDIR}" ]]; then
		rm -rf "${TMPDIR}"
	fi
}
trap cleanup EXIT

usage() {
	cat <<EOF
Usage: install-chat.sh [OPTIONS]

Install flowbot-chat (Chat Agent terminal client) from GitHub releases.

Options:
  --version TAG   Install a specific version (default: latest)
  --no-verify     Skip checksum verification
  --help          Show this help message

Examples:
  curl -fsSL https://raw.githubusercontent.com/flowline-io/flowbot/master/scripts/install-chat.sh | bash
  bash install-chat.sh --version v0.40
EOF
	exit 0
}

detect_platform() {
	local os arch

	case "$(uname -s)" in
		Linux)  os="linux" ;;
		Darwin) os="darwin" ;;
		MINGW*|CYGWIN*|MSYS*) os="windows" ;;
		*)
			error "Unsupported operating system: $(uname -s)"
			exit 1
			;;
	esac

	case "$(uname -m)" in
		x86_64 | amd64) arch="amd64" ;;
		aarch64 | arm64) arch="arm64" ;;
		*)
			error "Unsupported architecture: $(uname -m)"
			exit 1
			;;
	esac

	echo "${os}" "${arch}"
}

check_cmd() {
	if ! command -v "$1" &>/dev/null; then
		error "$1 is required but not installed"
		exit 1
	fi
}

download() {
	local url="$1"
	local dest="$2"
	curl -fsSL --retry 3 --retry-delay 2 -o "${dest}" "${url}"
}

verify_checksum() {
	local binary_path="$1"
	local asset_name="$2"

	local checksum_url
	if [[ "${VERSION}" == "latest" ]]; then
		checksum_url="https://github.com/${REPO}/releases/latest/download/flowbot_checksums.txt"
	else
		checksum_url="https://github.com/${REPO}/releases/download/${VERSION}/flowbot_checksums.txt"
	fi

	local checksum_file="${TMPDIR}/checksums.txt"
	info "Downloading checksums..."
	if ! download "${checksum_url}" "${checksum_file}"; then
		warn "Could not download checksum file, skipping verification"
		return 0
	fi

	local expected
	expected=$(awk -v asset="${asset_name}" '$2 == asset {print $1}' "${checksum_file}")

	if [[ -z "${expected}" ]]; then
		warn "No checksum entry found for ${asset_name}, skipping verification"
		return 0
	fi

	local actual
	if command -v shasum &>/dev/null; then
		actual=$(shasum -a 256 "${binary_path}" | awk '{print $1}')
	elif command -v sha256sum &>/dev/null; then
		actual=$(sha256sum "${binary_path}" | awk '{print $1}')
	else
		warn "No sha256 tool found (shasum/sha256sum), skipping verification"
		return 0
	fi

	if [[ "${expected}" != "${actual}" ]]; then
		error "Checksum verification failed"
		error "  expected: ${expected}"
		error "  actual:   ${actual}"
		exit 1
	fi

	info "Checksum verified"
}

install_binary() {
	local src="$1"
	local dest="${INSTALL_DIR}/${BINARY_NAME}"

	if [[ -w "${INSTALL_DIR}" ]]; then
		install -m 755 "${src}" "${dest}"
	else
		info "Writing to ${INSTALL_DIR} requires elevated permissions"
		if command -v sudo &>/dev/null; then
			sudo install -m 755 "${src}" "${dest}"
		else
			error "Elevated permissions required, but 'sudo' is not found"
			exit 1
		fi
	fi
}

main() {
	while [[ $# -gt 0 ]]; do
		case "$1" in
			--version)
				VERSION="$2"
				shift 2
				;;
			--no-verify)
				NO_VERIFY=true
				shift
				;;
			--help)
				usage
				;;
			*)
				error "Unknown option: $1"
				usage
				;;
		esac
	done

	check_cmd curl

	TMPDIR=$(mktemp -d)
	read -r OS ARCH < <(detect_platform)

	local asset_name="${CLI_ASSET}_${OS}_${ARCH}"
	if [[ "${OS}" == "windows" ]]; then
		asset_name="${asset_name}.exe"
	fi

	local download_url
	if [[ "${VERSION}" == "latest" ]]; then
		download_url="https://github.com/${REPO}/releases/latest/download/${asset_name}"
	else
		download_url="https://github.com/${REPO}/releases/download/${VERSION}/${asset_name}"
	fi

	local binary_path="${TMPDIR}/${BINARY_NAME}"

	info "Downloading flowbot-chat ${VERSION} (${OS}/${ARCH})..."
	if ! download "${download_url}" "${binary_path}"; then
		error "Failed to download from ${download_url}"
		error "Check that the version exists and your platform is supported"
		exit 1
	fi

	if [[ "${NO_VERIFY}" != true ]]; then
		verify_checksum "${binary_path}" "${asset_name}"
	else
		info "Checksum verification skipped (--no-verify)"
	fi

	install_binary "${binary_path}"

	info "flowbot-chat installed to ${INSTALL_DIR}/${BINARY_NAME}"

	if "${INSTALL_DIR}/${BINARY_NAME}" --help >/dev/null 2>&1; then
		info "Installation successful"
	else
		warn "Installation completed, but running '${BINARY_NAME} --help' failed"
	fi

	if ! command -v "${BINARY_NAME}" &>/dev/null; then
		warn "${INSTALL_DIR} may not be in your PATH"
		warn "Add 'export PATH=${INSTALL_DIR}:\$PATH' to your shell profile"
	fi
}

main "$@"
