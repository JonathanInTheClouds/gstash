#!/usr/bin/env bash
set -euo pipefail

# ============================================================
#  gstash installer
#  Usage: curl -fsSL https://raw.githubusercontent.com/JonathanInTheClouds/gstash/main/install.sh | bash
# ============================================================

REPO="JonathanInTheClouds/gstash"
BINARY="gstash"
INSTALL_DIR=""

# --- colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

info()    { echo -e "${CYAN}${BOLD}==>${RESET} $*"; }
success() { echo -e "${GREEN}${BOLD}✓${RESET} $*"; }
warn()    { echo -e "${YELLOW}${BOLD}!${RESET} $*"; }
error()   { echo -e "${RED}${BOLD}✗${RESET} $*" >&2; exit 1; }

# --- detect OS ---
detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    *)       error "Unsupported OS: $(uname -s). Only Linux and macOS are supported." ;;
  esac
}

# --- detect architecture ---
detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) error "Unsupported architecture: $(uname -m)." ;;
  esac
}

# --- pick install dir ---
detect_install_dir() {
  if [[ -w "/usr/local/bin" ]]; then
    echo "/usr/local/bin"
  elif [[ -n "${HOME:-}" && -d "$HOME/.local/bin" ]]; then
    echo "$HOME/.local/bin"
  elif [[ -n "${HOME:-}" ]]; then
    mkdir -p "$HOME/.local/bin"
    echo "$HOME/.local/bin"
  else
    error "Could not determine a writable install directory."
  fi
}

# --- check required tools ---
check_deps() {
  for cmd in curl sha256sum; do
    if ! command -v "$cmd" &>/dev/null; then
      # macOS uses shasum instead of sha256sum
      if [[ "$cmd" == "sha256sum" ]] && command -v shasum &>/dev/null; then
        continue
      fi
      error "Required tool not found: $cmd"
    fi
  done
}

# --- fetch latest version from GitHub ---
fetch_latest_version() {
  local version
  version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' \
    | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')

  if [[ -z "$version" ]]; then
    error "Could not fetch latest version from GitHub. Check your internet connection."
  fi
  echo "$version"
}

# --- checksum verification ---
verify_checksum() {
  local file="$1"
  local checksums_file="$2"
  local filename
  filename=$(basename "$file")

  info "Verifying checksum..."

  if command -v sha256sum &>/dev/null; then
    expected=$(grep "$filename" "$checksums_file" | awk '{print $1}')
    actual=$(sha256sum "$file" | awk '{print $1}')
  elif command -v shasum &>/dev/null; then
    expected=$(grep "$filename" "$checksums_file" | awk '{print $1}')
    actual=$(shasum -a 256 "$file" | awk '{print $1}')
  else
    warn "Could not verify checksum — neither sha256sum nor shasum found. Skipping."
    return 0
  fi

  if [[ "$expected" != "$actual" ]]; then
    error "Checksum mismatch!\n  Expected: $expected\n  Got:      $actual\nThe download may be corrupted."
  fi

  success "Checksum verified."
}

# ============================================================
#  Main
# ============================================================
main() {
  echo ""
  echo -e "${BOLD}  gstash installer${RESET}"
  echo -e "  A TUI for managing git stashes"
  echo ""

  check_deps

  OS=$(detect_os)
  ARCH=$(detect_arch)
  INSTALL_DIR=$(detect_install_dir)

  info "Detected platform: ${OS}/${ARCH}"

  # Fetch latest version
  info "Fetching latest release..."
  VERSION=$(fetch_latest_version)
  info "Latest version: ${VERSION}"

  # Build download URLs
  BINARY_NAME="${BINARY}-${OS}-${ARCH}"
  BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
  BINARY_URL="${BASE_URL}/${BINARY_NAME}"
  CHECKSUMS_URL="${BASE_URL}/checksums.txt"

  # Create temp dir
  TMP_DIR=$(mktemp -d)
  trap 'rm -rf "$TMP_DIR"' EXIT

  # Download binary
  info "Downloading ${BINARY_NAME}..."
  if ! curl -fsSL --progress-bar "$BINARY_URL" -o "${TMP_DIR}/${BINARY_NAME}"; then
    error "Failed to download binary from:\n  ${BINARY_URL}"
  fi

  # Download checksums
  info "Downloading checksums..."
  if ! curl -fsSL "$CHECKSUMS_URL" -o "${TMP_DIR}/checksums.txt"; then
    warn "Could not download checksums file. Skipping verification."
  else
    verify_checksum "${TMP_DIR}/${BINARY_NAME}" "${TMP_DIR}/checksums.txt"
  fi

  # Install
  info "Installing to ${INSTALL_DIR}/${BINARY}..."
  chmod +x "${TMP_DIR}/${BINARY_NAME}"

  if [[ -w "$INSTALL_DIR" ]]; then
    mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY}"
  else
    warn "Need sudo to write to ${INSTALL_DIR}"
    sudo mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY}"
  fi

  # Verify install
  if ! command -v "$BINARY" &>/dev/null; then
    warn "${INSTALL_DIR} may not be in your PATH."
    warn "Add this to your shell config (.zshrc / .bashrc):"
    warn "  export PATH=\"${INSTALL_DIR}:\$PATH\""
  fi

  echo ""
  success "gstash ${VERSION} installed successfully!"
  echo ""
  echo -e "  Run ${CYAN}${BOLD}gstash${RESET} from inside any git repository."
  echo -e "  Run ${CYAN}${BOLD}gstash --version${RESET} to confirm."
  echo ""
}

main "$@"