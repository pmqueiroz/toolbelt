#!/bin/sh
set -eu

repo="pmqueiroz/toolbelt"
base_url="${TOOLBELT_BASE_URL:-https://github.com/$repo/releases/latest/download}"
install_dir="${TOOLBELT_INSTALL_DIR:-$HOME/.local/bin}"

info() { printf '\033[38;5;212m==>\033[0m %s\n' "$1"; }
fail() { printf '\033[31merror:\033[0m %s\n' "$1" >&2; exit 1; }

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "unsupported OS $(uname -s), on windows run: irm https://toolbelt.pmqueiroz.dev/install.ps1 | iex" ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) fail "unsupported architecture $(uname -m)" ;;
esac

if command -v curl >/dev/null 2>&1; then
  download() { curl -fsSL "$1" -o "$2"; }
elif command -v wget >/dev/null 2>&1; then
  download() { wget -qO "$2" "$1"; }
else
  fail "curl or wget is required"
fi

if command -v sha256sum >/dev/null 2>&1; then
  sha256() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
  sha256() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
  fail "sha256sum or shasum is required"
fi

archive="tb_${os}_${arch}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

info "downloading $archive"
download "$base_url/$archive" "$tmp/$archive"
download "$base_url/checksums.txt" "$tmp/checksums.txt"

expected="$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")"
[ -n "$expected" ] || fail "$archive is missing from checksums.txt"
[ "$(sha256 "$tmp/$archive")" = "$expected" ] || fail "checksum mismatch for $archive"

tar -xzf "$tmp/$archive" -C "$tmp" tb
mkdir -p "$install_dir"
mv "$tmp/tb" "$install_dir/tb"
chmod +x "$install_dir/tb"
info "installed $("$install_dir/tb" version) to $install_dir"

has_brew() {
  command -v brew >/dev/null 2>&1 ||
    [ -x /opt/homebrew/bin/brew ] ||
    [ -x /usr/local/bin/brew ] ||
    [ -x /home/linuxbrew/.linuxbrew/bin/brew ] ||
    [ -x "$HOME/.linuxbrew/bin/brew" ]
}

if [ -z "${TOOLBELT_SKIP_BREW:-}" ] && ! has_brew; then
  info "installing homebrew"
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)" </dev/tty
fi

if [ "$(command -v tb 2>/dev/null)" = "$install_dir/tb" ]; then
  tb_cmd=tb
else
  tb_cmd="$install_dir/tb"
fi
info "run '$tb_cmd setup' to set up this machine"
