#!/bin/sh
set -eu

REPO="${REPO:-valpiks/backupctl}"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="${BINARY_NAME:-backupctl}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

download() {
  url="$1"
  out="$2"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$out"
    return
  fi

  if command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$out"
    return
  fi

  echo "error: curl or wget is required" >&2
  exit 1
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *) echo "error: unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) echo "error: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
}

latest_version() {
  tmp="${TMPDIR:-/tmp}/backupctl-release-$$.json"
  download "https://api.github.com/repos/$REPO/releases/latest" "$tmp"
  sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$tmp" | head -n 1
  rm -f "$tmp"
}

install_binary() {
  src="$1"
  dst="$2"

  if [ -w "$INSTALL_DIR" ]; then
    cp "$src" "$dst"
    chmod 0755 "$dst"
    return
  fi

  if command -v sudo >/dev/null 2>&1; then
    sudo mkdir -p "$INSTALL_DIR"
    sudo cp "$src" "$dst"
    sudo chmod 0755 "$dst"
    return
  fi

  echo "error: $INSTALL_DIR is not writable and sudo is not available" >&2
  exit 1
}

need_cmd uname
need_cmd tar
need_cmd sed

os="$(detect_os)"
arch="$(detect_arch)"

if [ "$VERSION" = "latest" ]; then
  VERSION="$(latest_version)"
fi

if [ -z "$VERSION" ]; then
  echo "error: could not resolve release version" >&2
  exit 1
fi

asset_version="${VERSION#v}"
archive="backupctl_${asset_version}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$VERSION/$archive"
tmpdir="$(mktemp -d)"

cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM

echo "Downloading $url"
download "$url" "$tmpdir/$archive"

tar -xzf "$tmpdir/$archive" -C "$tmpdir"

if [ ! -f "$tmpdir/$BINARY_NAME" ]; then
  echo "error: archive did not contain $BINARY_NAME" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR" 2>/dev/null || true
install_binary "$tmpdir/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"

echo "Installed $BINARY_NAME to $INSTALL_DIR/$BINARY_NAME"
"$INSTALL_DIR/$BINARY_NAME" version
