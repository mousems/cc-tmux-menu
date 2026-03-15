# Changelog

All notable changes to this project will be documented in this file.

## [0.2.0] - 2026-03-15

### Added
- First-time setup wizard — interactive configuration on initial launch
- Shell integration step in setup — auto-detects zsh/bash/fish and adds launch snippet
- `--help` / `-h` flag with usage info and keybindings
- `--setup` flag to re-run configuration wizard anytime
- Demo GIF in README

### Changed
- Quick start docs now show the setup wizard flow instead of manual steps

## [0.1.0] - 2026-03-14

### Added
- Interactive tmux session menu (arrow keys / jk navigation)
- AI-powered session preview via OpenRouter (one-line summary + suggested name)
- Session management: attach, rename, kill
- Random `adjective-noun` session naming
- Remote SSH with auto-connectivity detection (direct / DNAT tunnel / offline)
- Preview cache with 30-min TTL
- Config validation with warnings
- TOML and legacy shell config format support
- `--version` / `-v` flag
- GitHub Actions CI (linux/darwin x amd64/arm64)
- goreleaser for multi-platform releases
- Homebrew tap (`brew install mousems/tap/cc-tmux-menu`)
