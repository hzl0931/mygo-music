#!/usr/bin/env bash
# Builds the Windows app and its NSIS installer into build/windows-amd64.
#
#   scripts/build-windows.sh            windows/amd64
#   scripts/build-windows.sh --arm64    windows/amd64 and windows/arm64
#   scripts/build-windows.sh --portable the app alone, without the installer
#
# mygo.config.ts is evaluated with Bun or Node.js 22.6+. mygo build downloads
# NSIS on Windows when it is not installed. The installer is unsigned, so
# SmartScreen may warn about it on another machine.
set -euo pipefail

cd "$(dirname "$0")/.."

platform=windows/amd64
installer=1
for arg in "$@"; do
	case "$arg" in
	--arm64) platform=windows/amd64,windows/arm64 ;;
	--portable) installer=0 ;;
	-h | --help)
		sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//'
		exit 0
		;;
	*)
		echo "unknown option: $arg" >&2
		exit 2
		;;
	esac
done

if ! command -v bun >/dev/null 2>&1 && ! command -v node >/dev/null 2>&1; then
	echo "evaluating mygo.config.ts needs Bun or Node.js 22.6 or later" >&2
	exit 1
fi

go build ./...

args=(-platform "$platform")
if [[ $installer == 0 ]]; then
	args+=(-skip-installer)
fi
go tool mygo build "${args[@]}"

for f in build/windows-*/mygo-music-setup-*.exe; do
	[[ -e $f ]] || continue
	sha256sum "$f"
done
