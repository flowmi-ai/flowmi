# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Flowmi CLI (`flowmi` or `fm`) — a Go command-line tool for OAuth2 PKCE authentication and API interactions with the Flowmi platform.

## Commands

```bash
go build -o bin/flowmi .                        # Build
go test ./... -v -race -cover                   # All tests
go test ./internal/auth/ -v -run TestGeneratePKCE  # Single test
```

Version info is injected via ldflags (`version`, `commit`, `date` in `cmd/version.go`).

## Architecture

Cobra + Viper CLI. Entry: `main.go` → `cmd.Execute()`.

```
cmd/           Cobra commands (login, whoami, note, configure, version)
internal/
  auth/        OAuth2 PKCE generation, token exchange, local callback server
  api/         REST API client (envelope response format)
  config/      XDG-compliant paths (~/.config/flowmi/), TOML credentials (0600)
  ui/          lipgloss terminal styles
```

### Key Patterns

- **Two login flows**: browser OAuth2 PKCE (default) and direct password login (`--email`/`--password` for CI/CD). Both use PKCE.
- **Output format**: all display commands support `-o text|json|table` via Viper global flag.
- **Auth state**: commands check `viper.GetString("access_token")` — credentials are loaded into Viper defaults at init from `credentials.toml`.
- **API envelope**: server responses use `{"success": bool, "data": ..., "error": {"code": "...", "message": "..."}}`. The `api.Client.do()` method handles unwrapping.
- **Binary alias**: supports both `flowmi` and `fm` — `cmd/root.go` adapts `Use` field based on `os.Args[0]`.
- **Config precedence**: flags → env vars (`FLOWMI_` prefix) → config.toml → credentials.toml defaults → hardcoded defaults (`auth.flowmi.ai`, `api.flowmi.ai`).

## CLI Design

Follow GitHub CLI (`gh`) as the design reference for all command-line interface decisions.

**Reference:** https://cli.github.com/manual/

**Before designing any new command or flag**, run `gh <command> --help` locally to study how `gh` handles similar functionality, then mimic its patterns.

Key conventions to follow:
- **`noun verb` structure**: `fm note list`, `fm note create` (like `gh issue list`, `gh pr create`)
- **Flags over positional args**: use named flags for parameters (`--title`, `--body`), not positional arguments
- **Short flags**: provide single-letter aliases for common flags (`-t` for `--title`, `-o` for `--output`)
- **`--json` field selection**: support `--json field1,field2` for structured output
- **Consistent verbs**: `list`, `view`, `create`, `edit`, `delete` (not `show`/`get`/`add`/`update`/`remove`)
- **Interactive prompts**: when required flags are missing, prompt interactively instead of erroring
- **`--web` flag**: open the resource in browser where applicable

## Flowmi Ecosystem

Flowmi is an OAuth2 PKCE auth ecosystem with three independent repos:
- **flowmi** (Go CLI) → **web** (SvelteKit auth server) → **server** (Go REST API + Postgres/Redis)
- Flow: CLI generates PKCE pair → opens browser to web `/authorize` → user logs in → redirect with auth code → CLI exchanges code for tokens

## GitHub

All Flowmi repos are under the **humid888** account. Run `gh auth switch` to humid888 before pushing, creating PRs, etc.

## Decision Making

When making technical decisions, follow community best practices over personal preference:
1. **Research first** — Search HN, Lobsters, SO, Reddit for real-world discussions before deciding
2. **Don't repeat known mistakes** — If the internet has lessons learned, don't rediscover them
3. **Be honest about consensus** — If a direction conflicts with community consensus, say so clearly
