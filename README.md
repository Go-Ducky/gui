# GoDucky GUI 🦆

The desktop edition of [GoDucky](https://github.com/Go-Ducky/cli) — an AI coding
agent that reads, writes, edits, searches, and runs commands in a project folder.
A chat desktop app built with **native toolkits** (no webview runtime).

The backend reuses the exact same runtime as the CLI: agent tool loop, providers,
sessions, and setup, exposed through a UI-agnostic engine
(`internal/native`) that any frontend can drive.

## Features

- Sidebar session list with New Chat, rename, delete, and auto-save on quit
- Streaming responses with live tool execution blocks (approve/deny built-in)
- Providers: Ollama (local), Groq, OpenAI, OpenAI-compatible, Anthropic, Gemini, OpenRouter
- Onboarding wizard: run fully local with Ollama or use a cloud API key
- Working-directory picker, provider/model switchers, and settings inline
- Theming: light / dark / system

## Frontends

| Frontend | Toolkit | Build tag |
| --- | --- | --- |
| GTK4 | gotk4 (`diamondburned/gotk4`) | `gtk` |
| Qt 6 | miqt (`mappu/miqt`) | `qt` |
| Cocoa (macOS) | planned | — |
| Win32 (Windows) | planned | — |

## Install

| Platform | Package |
| --- | --- |
| Linux (Arch) | `paru -S goducky` (AUR), or the release tarball |
| Linux (GTK4) | `goducky-linux-amd64` from [Releases](https://github.com/Go-Ducky/gui/releases), or the `.tar.gz` (unpacks into `/usr`) |

macOS and Windows installers resume once the Cocoa and Win32 backends land.

## Build from source

Requirements: Go 1.27+, a C toolchain, and the GTK4 or Qt 6 dev libraries
(e.g. on Arch: `gtk4` or `qt6-base`).

```sh
task build:gtk   # or: go build -tags gtk -o bin/goducky .
task run:gtk

task build:qt    # or: go build -tags qt -o bin/goducky .
task run:qt
```

## Releases

Pushing to `main` builds a `dev-<sha>` prerelease; pushing a `v*` tag builds a
proper release. Releases currently ship the native GTK4 Linux binary and a
tarball.

## Project structure

- `internal/guiservice/` — the shared backend that wraps the agent runtime
- `internal/native/` — UI-agnostic engine plus GTK4 and Qt 6 frontends
- `internal/{agent,provider,config,session,setup}` — shared with the CLI
- `packaging/linux/` — .desktop entry and app icon
- `packaging/aur/` — AUR PKGBUILD
- `.github/workflows/` — CI + release pipeline

## License

MIT