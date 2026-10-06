#!/bin/sh
# End-to-end test for site/install.sh against locally built release assets.
#
#   sh tests/install/run.sh [asset-dir]   (default: bin, as written by `make cross-compile`)
#
# Runs the installer under dash (strict POSIX) when available, else sh.
set -eu

root="$(cd "$(dirname "$0")/../.." && pwd)"
assets="$(cd "${1:-$root/bin}" && pwd)"
script="$root/site/install.sh"
shell="$(command -v dash || command -v sh)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

pass() { printf 'ok   %s\n' "$1"; }
die() {
	printf 'FAIL %s\n' "$1" >&2
	[ -f "$work/out" ] && sed 's/^/     /' "$work/out" >&2
	exit 1
}

# run <name> [VAR=value ...]: installs into $work/<name>, output in $work/out
run() {
	name="$1"
	shift
	env JOKATEKO_INSTALL_DIR="$work/$name" "$@" "$shell" "$script" >"$work/out" 2>&1
}

[ -f "$assets/checksums.txt" ] || die "no checksums.txt in $assets; run make cross-compile"

# 1. Happy path: verifies, installs, the binary runs
run ok JOKATEKO_DOWNLOAD_URL="file://$assets" || die "install from $assets"
"$work/ok/jokateko" version >/dev/null || die "installed binary does not run"
grep -q 'checksum verified' "$work/out" || die "no checksum confirmation"
pass "installs and verifies the binary"

# 2. Reinstall over an existing binary
run ok JOKATEKO_DOWNLOAD_URL="file://$assets" || die "reinstall over existing binary"
pass "reinstalls over an existing binary"

# 3. Checksum mismatch: fails, installs nothing
mkdir "$work/bad"
cp "$assets"/jokateko-* "$work/bad/"
zeros=0000000000000000000000000000000000000000000000000000000000000000
awk -v z="$zeros" '{ print z "  " $2 }' "$assets/checksums.txt" >"$work/bad/checksums.txt"
if run mismatch JOKATEKO_DOWNLOAD_URL="file://$work/bad"; then die "mismatch accepted"; fi
grep -q 'checksum mismatch' "$work/out" || die "mismatch message"
[ ! -e "$work/mismatch/jokateko" ] || die "mismatch still installed a binary"
pass "rejects a checksum mismatch"

# 4. Missing checksum entry
: >"$work/bad/checksums.txt"
if run missing JOKATEKO_DOWNLOAD_URL="file://$work/bad"; then die "missing entry accepted"; fi
grep -q 'has no entry' "$work/out" || die "missing entry message"
pass "rejects a missing checksum entry"

# 5. Unsupported OS (fake uname first on PATH)
mkdir "$work/fakebin"
printf '#!/bin/sh\necho FreeBSD\n' >"$work/fakebin/uname"
chmod +x "$work/fakebin/uname"
if run os PATH="$work/fakebin:$PATH" JOKATEKO_DOWNLOAD_URL="file://$assets"; then die "unsupported OS accepted"; fi
grep -q "unsupported OS 'FreeBSD'" "$work/out" || die "unsupported OS message"
pass "rejects an unsupported OS"

# 6. Malformed version
if run version JOKATEKO_VERSION="../../evil"; then die "malformed version accepted"; fi
grep -q 'must be a release tag' "$work/out" || die "malformed version message"
pass "rejects a malformed JOKATEKO_VERSION"

printf 'install.sh: all checks passed (%s)\n' "$shell"
