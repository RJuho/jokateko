#!/bin/sh
# Jokateko installer for Linux and macOS.
#
#   curl -fsSL https://jokateko.dev/install.sh | sh
#
# Downloads the release binary for this OS/architecture from GitHub Releases,
# verifies its SHA-256 against the release's checksums.txt and installs it
# without sudo. Windows: download the .exe from the Releases page.
#
# Environment:
#   JOKATEKO_VERSION       release tag to install, e.g. v0.1.0 (default: latest stable)
#   JOKATEKO_INSTALL_DIR   target directory (default: ~/.local/bin)
#   JOKATEKO_DOWNLOAD_URL  base URL holding the release assets (mirrors, testing)

# Everything runs inside main(), called on the last line, so a truncated
# download never executes a partial script.
set -eu

REPO_URL="https://github.com/RJuho/jokateko"

say() {
	printf 'jokateko: %s\n' "$1"
}

fail() {
	printf 'jokateko: error: %s\n' "$1" >&2
	exit 1
}

detect_os() {
	case "$(uname -s)" in
	Linux) echo linux ;;
	Darwin) echo darwin ;;
	*) fail "unsupported OS '$(uname -s)'; download a binary from $REPO_URL/releases" ;;
	esac
}

detect_arch() {
	arch="$(uname -m)"
	# A shell under Rosetta reports x86_64 on Apple silicon; prefer the native binary
	if [ "$arch" = x86_64 ] && [ "$(sysctl -n sysctl.proc_translated 2>/dev/null || echo 0)" = 1 ]; then
		arch=arm64
	fi
	case "$arch" in
	x86_64 | amd64) echo amd64 ;;
	aarch64 | arm64) echo arm64 ;;
	*) fail "unsupported architecture '$arch'; download a binary from $REPO_URL/releases" ;;
	esac
}

download() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL --proto '=https,file' --proto-redir '=https' -o "$2" "$1" ||
			fail "download failed: $1"
	elif command -v wget >/dev/null 2>&1; then
		wget -q --https-only -O "$2" "$1" || fail "download failed: $1"
	else
		fail "curl or wget is required"
	fi
}

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		fail "sha256sum or shasum is required to verify the download"
	fi
}

main() {
	os="$(detect_os)"
	arch="$(detect_arch)"
	asset="jokateko-$os-$arch"
	install_dir="${JOKATEKO_INSTALL_DIR:-$HOME/.local/bin}"
	version="${JOKATEKO_VERSION:-latest}"

	if [ -n "${JOKATEKO_DOWNLOAD_URL:-}" ]; then
		base="$JOKATEKO_DOWNLOAD_URL"
	elif [ "$version" = latest ]; then
		base="$REPO_URL/releases/latest/download"
	else
		case "$version" in
		v[0-9]*) ;;
		*) fail "JOKATEKO_VERSION must be a release tag like v0.1.0, got '$version'" ;;
		esac
		base="$REPO_URL/releases/download/$version"
	fi

	tmp="$(mktemp -d)"
	trap 'rm -rf "$tmp"' EXIT
	trap 'exit 1' HUP INT TERM

	say "downloading $asset ($version)"
	download "$base/$asset" "$tmp/$asset"
	download "$base/checksums.txt" "$tmp/checksums.txt"

	expected="$(awk -v f="$asset" '$2 == f || $2 == "*" f { print $1; exit }' "$tmp/checksums.txt")"
	[ -n "$expected" ] || fail "checksums.txt has no entry for $asset"
	actual="$(sha256_of "$tmp/$asset")"
	[ "$expected" = "$actual" ] ||
		fail "checksum mismatch for $asset (expected $expected, got $actual); nothing was installed"
	say "checksum verified"

	mkdir -p "$install_dir"
	chmod 0755 "$tmp/$asset"
	# Copy next to the target, then rename: replacing a running binary stays safe
	cp "$tmp/$asset" "$install_dir/.jokateko.tmp.$$"
	mv -f "$install_dir/.jokateko.tmp.$$" "$install_dir/jokateko"
	say "installed $install_dir/jokateko"
	"$install_dir/jokateko" version || fail "the installed binary does not run on this system"

	case ":$PATH:" in
	*":$install_dir:"*)
		found="$(command -v jokateko 2>/dev/null || true)"
		if [ -n "$found" ] && [ "$found" != "$install_dir/jokateko" ]; then
			say "note: $found comes first on PATH and shadows this install"
		fi
		;;
	*)
		say "$install_dir is not on your PATH; add this line to your shell profile:"
		printf '\n    export PATH="%s:$PATH"\n\n' "$install_dir"
		;;
	esac
	say "get started: cd your-project && jokateko init && jokateko serve"
}

main "$@"
