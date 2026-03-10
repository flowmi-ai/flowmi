# Flowmi CLI

**flowmi** — notes, drive, email, tables, search, and more from your terminal.

[![CI](https://github.com/flowmi-ai/flowmi/actions/workflows/ci.yml/badge.svg)](https://github.com/flowmi-ai/flowmi/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Installation

```bash
curl -fsSL https://flowmi.ai/install | bash
```

### Homebrew

```bash
brew install flowmi-ai/tap/flowmi # macOS and Linux (recommended, always up to date)
```

### Go

```bash
go install github.com/flowmi-ai/flowmi@latest
```

Requires Go 1.25+.

### Download binary

Download the latest release from the [Releases](https://github.com/flowmi-ai/flowmi/releases) page.

### Build from source

```bash
git clone https://github.com/flowmi-ai/flowmi.git
cd flowmi
make build
cp bin/flowmi ~/.local/bin/
```

Requires Go 1.25+.

## Quick start

```bash
# Authenticate (opens browser for OAuth2 PKCE flow)
flowmi auth login

# Or use email/password for CI/CD
flowmi auth login --email user@example.com --password '...'

# Check auth status
flowmi auth status

# Create a note
flowmi note create --subject "Hello" --content "My first note"

# List notes
flowmi note list

# Upload a file
flowmi drive upload ./report.pdf

# Send an email
flowmi email send --to user@example.com --subject "Hi" --text "Hello from Flowmi"

# Web search
flowmi search "golang best practices"

# Scrape a webpage
flowmi scrape https://example.com
```

## Commands

```
flowmi auth login|status                                   Authentication
flowmi note list|create|view|edit|delete|trash|restore     Notes
flowmi drive list|upload|download|view|delete|trash|restore   Cloud drive
flowmi table list|create|view|edit|delete|trash|restore    Tables
flowmi table field add|edit|delete                         Table fields
flowmi table row list|create|view|edit|delete|query|trash|restore   Table rows
flowmi email send|list|view|delete|trash|restore           Email
flowmi email mailbox list|create|edit|delete               Mailboxes
flowmi search [web|images|news]                            Web search
flowmi scrape <url>                                        Web scraping
flowmi config set|get|list                                 Configuration
flowmi update                                              Update to latest version
flowmi completion bash|zsh|fish|powershell                 Shell completion
flowmi version                                             Version info
flowmi options                                             Show global flags
```

Use `flowmi <command> --help` for detailed usage of any command.

## Output formats

All commands support `--output` (`-o`) to control output format:

```bash
flowmi note list -o json     # JSON output
flowmi note list -o table    # Table output
flowmi note list -o text     # Text output (default)
```

## Configuration

Configuration is stored in `~/.config/flowmi/` (XDG-compliant):

- `config.toml` — server URLs and preferences
- `credentials.toml` — tokens and API keys (0600 permissions)

```bash
flowmi config list              # Show all config values
flowmi config set api_key sk-...  # Set a value
flowmi config get api_server_url  # Get a value
```

Environment variables with the `FLOWMI_` prefix override config file values (e.g., `FLOWMI_API_KEY`).

## Shell completion

```bash
# Bash
source <(flowmi completion bash)

# Zsh
flowmi completion zsh > "${fpath[1]}/_flowmi"

# Fish
flowmi completion fish | source
```

## License

[Apache License 2.0](LICENSE)
