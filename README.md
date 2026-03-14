# cc-tmux-menu

> Interactive tmux session manager with AI-powered previews.
> Built for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) multi-session workflows.

[![Go](https://img.shields.io/github/go-mod/go-version/mousems/cc-tmux-menu)](https://go.dev/)
[![Release](https://img.shields.io/github/v/release/mousems/cc-tmux-menu)](https://github.com/mousems/cc-tmux-menu/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

<!-- TODO: replace with actual demo GIF -->
<!-- ![demo](demo.gif) -->

## Features

- **Session menu** — navigate with arrow keys / jk, enter to attach, q to quit
- **AI preview** — each session gets a one-line summary of what it's working on (via [OpenRouter](https://openrouter.ai))
- **AI rename** — suggested session names based on content (e.g. `API-Refactor`, `DB-Debug`)
- **Session management** — attach, rename, kill from a single menu
- **Random naming** — new sessions get `adjective-noun` names (e.g. `swift-fox`, `calm-reef`)
- **Remote SSH** — optional SSH shortcut with auto-connectivity detection (direct / DNAT tunnel / offline)
- **Works without AI** — no API key? Previews fall back to showing the last output line

## Install

### Homebrew (macOS / Linux)

```bash
brew install mousems/tap/cc-tmux-menu
```

### Download binary

Download from [Releases](https://github.com/mousems/cc-tmux-menu/releases/latest), extract, and move to your PATH:

```bash
# macOS (Apple Silicon)
curl -sL https://github.com/mousems/cc-tmux-menu/releases/latest/download/cc-tmux-menu_darwin_arm64.tar.gz | tar xz
mv cc-tmux-menu ~/.local/bin/

# Linux (amd64)
curl -sL https://github.com/mousems/cc-tmux-menu/releases/latest/download/cc-tmux-menu_linux_amd64.tar.gz | tar xz
mv cc-tmux-menu ~/.local/bin/
```

### From source

```bash
go install github.com/mousems/cc-tmux-menu/cmd/cc-tmux-menu@latest
```

Or clone and build:

```bash
git clone https://github.com/mousems/cc-tmux-menu.git
cd cc-tmux-menu
make install   # builds and copies to ~/.local/bin/
```

## Quick start

```bash
# 1. Run it
cc-tmux-menu

# 2. (Optional) Add OpenRouter API key for AI previews
mkdir -p ~/.config/cc-tmux-menu
cat > ~/.config/cc-tmux-menu/config.toml << 'EOF'
openrouter_api_key = "sk-or-v1-..."
EOF

# 3. (Optional) Auto-launch on new terminal
echo '[[ -z "$TMUX" && -x "$(command -v cc-tmux-menu)" ]] && cc-tmux-menu' >> ~/.zshrc
```

## Usage

```
tmux sessions  (↑↓/jk select, Enter confirm, q quit)

  ▸ Remote SSH     mouse@server ● direct
    New Claude     new session (claude)
    Resume Claude  continue last (claude -r)
    Shell          plain shell
    swift-fox      2h ago
    calm-reef      5h ago (attached)
    Exit           stay in shell
    Disconnect     exit SSH

Preview: Discussing API integration for the new feature
```

Select a session to manage it:

```
Session: swift-fox  (←/h back, ↑↓/jk select, Enter confirm)

  ▸ Attach    connect to swift-fox
    Rename    rename session
    Kill      kill session
```

Renaming pre-fills the AI-suggested name — edit inline or press Esc to cancel.

## Configuration

Config file locations (first found wins):

1. `{binary dir}/config.toml`
2. `~/.config/cc-tmux-menu/config.toml`

Full config reference:

```toml
# OpenRouter API key (optional — AI features disabled without it)
# Get one at https://openrouter.ai/keys
openrouter_api_key = ""

# AI model for summaries (cheap & fast recommended)
openrouter_model = "google/gemini-2.0-flash-001"

# Working directory for new Claude Code sessions
work_dir = "~"

# Language for AI summaries (en, zh-TW, ja, ko, ...)
summary_lang = "en"

# Command to launch Claude Code
claude_cmd = "claude"

# Preview cache TTL in seconds (default: 30 min)
preview_cache_ttl = 1800

# Remote SSH (optional — adds SSH shortcut to menu)
remote_host = ""
remote_user = ""
remote_gateway_lan = ""
remote_gateway_wan = ""
remote_gateway_wan_port = 0
```

## How AI preview works

1. On menu open, each session's last 150 lines are captured in the background
2. ANSI codes and Claude Code UI elements are stripped
3. Content is sent to OpenRouter for a one-line summary + suggested name
4. Results are cached for 30 minutes (in `$TMPDIR/cc-tmux-menu-preview-cache/`)
5. While loading, sessions show "loading..." — you can still navigate

**Cost**: ~2K input tokens per summary. With `gemini-2.0-flash-001`, expect < $0.001 per call.

## Remote SSH

When `remote_host` is configured, the menu shows a **Remote SSH** item with live connectivity status:

| Status | Meaning | Connection |
|--------|---------|------------|
| `● direct` | LAN gateway reachable | `ssh user@host` |
| `● tunnel` | WAN gateway reachable | `ssh -p port user@gateway` |
| `● offline` | Neither reachable | Blocked |

This is useful when a router/gateway uses DNAT port forwarding to expose internal servers. Connectivity is checked via TCP dial (no ICMP/root required).

<details>
<summary>Example: DNAT through a MikroTik gateway</summary>

```
WAN                              LAN
┌──────────┐   DNAT 2020→22   ┌──────────────┐
│ Laptop   │──── gateway:2020 ──→│ Server       │
└──────────┘                   └──────────────┘
                ┌──────────┐
                │ Gateway  │  LAN: 10.0.0.1
                │          │  WAN: 203.0.113.1
                └──────────┘
```

```toml
remote_host = "10.0.0.50"
remote_user = "user"
remote_gateway_lan = "10.0.0.1"
remote_gateway_wan = "203.0.113.1"
remote_gateway_wan_port = 2022
```

</details>

## Requirements

- **tmux** — the only hard dependency
- **OpenRouter API key** — optional, for AI features ([get one here](https://openrouter.ai/keys))

## License

[MIT](LICENSE)
