# Flowmi CLI

**flowmi** (`fm`) — notes, drive, email, tables, search, and more from your terminal.

[![CI](https://github.com/flowmi-ai/flowmi/actions/workflows/ci.yml/badge.svg)](https://github.com/flowmi-ai/flowmi/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Installation

```bash
curl -fsSL https://flowmi.ai/install | bash
```

### Homebrew <!-- macOS and Linux (recommended, always up to date) -->

```bash
brew install flowmi-ai/tap/flowmi
```

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
fm auth login

# Or use email/password for CI/CD
fm auth login --email user@example.com --password '...'

# Check auth status
fm auth status

# Create a note
fm note create --subject "Hello" --content "My first note"

# List notes
fm note list

# Upload a file
fm drive upload ./report.pdf

# Send an email
fm email send --to user@example.com --subject "Hi" --body "Hello from Flowmi"

# Web search
fm search "golang best practices"

# Scrape a webpage
fm scrape https://example.com
```

## Commands

```
fm auth login|status              Authentication
fm note list|create|view|edit|delete|trash|restore   Notes
fm drive list|upload|download|view|delete             Cloud drive
fm table list|create|view|edit|delete                 Tables
fm table field add|edit|delete                        Table fields
fm table row list|create|view|edit|delete|query       Table rows
fm email send|list|view|delete                        Email
fm email mailbox list|create|edit|delete              Mailboxes
fm search [web|images|news]                           Web search
fm scrape <url>                                       Web scraping
fm config set|get|list                                Configuration
fm completion bash|zsh|fish|powershell                Shell completion
fm version                                            Version info
```

Use `fm <command> --help` for detailed usage of any command.

## Output formats

All commands support `--output` (`-o`) to control output format:

```bash
fm note list -o json     # JSON output
fm note list -o table    # Table output
fm note list -o text     # Text output (default)
```

## Configuration

Configuration is stored in `~/.config/flowmi/` (XDG-compliant):

- `config.toml` — server URLs and preferences
- `credentials.toml` — tokens and API keys (0600 permissions)

```bash
fm config list              # Show all config values
fm config set api_key sk-...  # Set a value
fm config get api_server_url  # Get a value
```

Environment variables with the `FLOWMI_` prefix override config file values (e.g., `FLOWMI_API_KEY`).

## Shell completion

```bash
# Bash
source <(fm completion bash)

# Zsh
fm completion zsh > "${fpath[1]}/_flowmi"

# Fish
fm completion fish | source
```

## License

[Apache License 2.0](LICENSE)
