---
name: build-windows-installer
description: Build MyGo Music's Windows exe and NSIS installer into build/windows-amd64. Use when the user asks to build, package, or produce a Windows installer/exe for this fork.
---

# Build the Windows installer

This repository is the MyGo Music fork, including the built-in furigana work;
build from its root, not from `../mygo-furigana-plugin`.

## Build

```bash
scripts/build-windows.sh
```

Options:

- `--arm64` — also build `windows/arm64`.
- `--portable` — the app alone, without the installer.

The script runs `go build ./...`, then
`go tool mygo build -platform windows/amd64`, and prints the SHA-256 of each
installer.

## Output

- `build/windows-amd64/mygo-music-setup-<version>-amd64.exe` — the installer.
- `build/windows-amd64/MyGo Music.exe` — the portable app.
- `build/SHA256SUMS` — checksums written by `mygo build`.

The version comes from `mygo.config.ts` (`0.1.25-furigana.1` at the time of
writing). Do not bump it unless the user asks.

## Requirements and caveats

- `mygo.config.ts` is evaluated with Bun or Node.js 22.6 or later. Node is
  installed here; Bun usually is not.
- `mygo build` downloads NSIS on Windows when `makensis` is missing.
- The installer is unsigned, so SmartScreen may warn on another machine.
- No updater signing happens without `MYGO_UPDATER_PRIVATE_KEY`; that is
  expected and does not affect the installer.
- If the build fails, check `go build ./...` and the Node version before
  touching the build flags.

## When done

Report the installer path, its size and SHA-256, and whether the build was
signed. Link the file with a relative path:

`[mygo-music-setup-<version>-amd64.exe](build/windows-amd64/...)`
