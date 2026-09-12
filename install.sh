#!/bin/sh

set -eu

binary_name="homelabc"
install_dir="${INSTALL_DIR:-/usr/local/bin}"
repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
build_dir=$(mktemp -d "${TMPDIR:-/tmp}/homelabc-install.XXXXXX")

cleanup() {
	rm -rf -- "$build_dir"
}
trap cleanup EXIT HUP INT TERM

if ! command -v go >/dev/null 2>&1; then
	echo "error: Go is required to build $binary_name" >&2
	exit 1
fi

echo "Building $binary_name..."
(
	cd "$repo_dir"
	go build -trimpath -o "$build_dir/$binary_name" .
)

if ! mkdir -p "$install_dir" 2>/dev/null; then
	echo "error: cannot create install directory $install_dir" >&2
	echo "try: sudo INSTALL_DIR=$install_dir $repo_dir/install.sh" >&2
	exit 1
fi

if ! install -m 0755 "$build_dir/$binary_name" "$install_dir/$binary_name"; then
	echo "error: cannot install $install_dir/$binary_name" >&2
	echo "try: sudo INSTALL_DIR=$install_dir $repo_dir/install.sh" >&2
	exit 1
fi

echo "Installed $install_dir/$binary_name"
