#!/usr/bin/env sh

set -eu

version="${REVIEWER_VERSION:-latest}"
prefix="${REVIEWER_PREFIX:-$HOME/.local/bin}"
os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os:$arch" in
    darwin:x86_64) target_os=darwin; target_arch=amd64 ;;
    darwin:arm64) target_os=darwin; target_arch=arm64 ;;
    linux:x86_64) target_os=linux; target_arch=amd64 ;;
    linux:aarch64|linux:arm64) target_os=linux; target_arch=arm64 ;;
    *) echo "reviewer: unsupported platform $os/$arch" >&2; exit 1 ;;
esac

if [ "$version" = latest ]; then
    api_url=https://api.github.com/repos/dapi/reviewer/releases/latest
    version="$(curl -fsSL "$api_url" | awk -F'"' '/"tag_name"/ { sub(/^v/, "", $4); print $4; exit }')"
fi

if ! printf '%s' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "reviewer: invalid version" >&2
    exit 1
fi

base_url="https://github.com/dapi/reviewer/releases/download/v$version"
archive="reviewer_${version}_${target_os}_${target_arch}.tar.gz"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT HUP INT TERM

curl -fsSL "$base_url/$archive" -o "$tmpdir/$archive"
curl -fsSL "$base_url/SHA256SUMS" -o "$tmpdir/SHA256SUMS"
if command -v sha256sum >/dev/null 2>&1; then
    (cd "$tmpdir" && grep "  $archive$" SHA256SUMS | sha256sum -c -)
else
    (cd "$tmpdir" && grep "  $archive$" SHA256SUMS | shasum -a 256 -c -)
fi
mkdir -p "$prefix"
tar -xzf "$tmpdir/$archive" -C "$tmpdir"
install -m 0755 "$tmpdir/reviewer" "$prefix/reviewer"
printf 'Installed reviewer v%s to %s/reviewer\n' "$version" "$prefix"
