<<<<<<< HEAD
# gui
GoDucky GUI Application
=======
# GoDucky GUI 🦆

The desktop edition of [GoDucky](https://github.com/Go-Ducky/cli) — an AI coding
agent that reads, writes, edits, searches, and runs commands in a project folder.
This is a modern chat desktop app (like opencode) for Linux, macOS, and Windows,
powered by [Wails v3](https://v3.wails.io/).

The backend reuses the exact same runtime as the CLI: agent tool loop, providers,
sessions, and setup, wrapped in a Wails service the frontend calls directly.

## Features

- Sidebar session list with New Chat, rename, delete, and auto-save on quit
- Streaming responses with live tool execution blocks (approve/deny built-in)
- Providers: Ollama (local), Groq, OpenAI, OpenAI-compatible, Anthropic, Gemini, OpenRouter
- Onboarding wizard: run fully local with Ollama or use a cloud API key
- Working-directory picker, provider/model switchers, and settings inline
- Cross-platform installers built entirely in GitHub Actions

## Install

| Platform | Package |
| --- | --- |
| Linux (AppImage) | download `goducky-x86_64.AppImage` from [Releases](https://github.com/Go-Ducky/gui/releases), run it |
| Linux (Debian/Ubuntu) | `sudo apt install ./goducky-<version>-amd64.deb` |
| Linux (Fedora/RHEL) | `sudo dnf install goducky-<version>-x86_64.rpm` |
| Linux (Arch) | `paru -S goducky` (AUR), or install the bundled `goducky-<version>-1-x86_64.pkg.tar.zst` |
| macOS | download `goducky-<version>-universal.dmg` |
| Windows | download `goducky-<version>-setup.exe` |

## Build from source

Requirements: Go 1.25+, Node.js (or bun), and a C toolchain with the
[Wails Linux/macOS/Windows system libraries](https://v3.wails.io/getting-started/installation/).

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@latest

wails3 task build    # production binary in bin/
wails3 dev           # hot-reload development mode
```

Installers per platform:

```sh
wails3 task linux:package      # AppImage + .deb + .rpm + Arch pkg
wails3 task darwin:package:dmg # macOS .dmg
wails3 task windows:package    # Windows NSIS setup.exe
```

## Releases

Pushing to `main` builds a `dev-<sha>` prerelease; pushing a `v*` tag builds a
proper release. Each release ships binaries and installers for all platforms.

## Project structure

- `main.go` — Wails app setup (window, closing auto-save)
- `internal/guiservice/` — the bound backend that wraps the agent runtime
- `internal/{agent,provider,config,session,setup}` — shared with the CLI
- `frontend/` — TypeScript + Vite chat UI
- `build/` — Wails per-OS packaging (config, nfpm, AppImage, NSIS, DMG)
- `.github/workflows/` — CI + release pipeline

## License

MIT
>>>>>>> 0ee89dc (GoDucky GUI: Wails v3 chat desktop app for the GoDucky agent)
