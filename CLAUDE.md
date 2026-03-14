# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Flowmi CLI (`flowmi` or `fm`) — a Go command-line tool for OAuth2 PKCE authentication and API interactions with the Flowmi platform.

## Commands

```bash
make build                                      # Build to bin/flowmi (with ldflags)
make dev                                        # Dev build (no symbol stripping)
make test                                       # All tests (go test ./... -v -race -cover)
make lint                                       # golangci-lint run
make fmt                                        # gofmt -s -l -w .
make vet                                        # go vet ./...
go test ./internal/auth/ -v -run TestGeneratePKCE  # Single test
```

Version info is injected via ldflags (`version`, `commit`, `date` in `cmd/version.go`).

**After every code change**, run `make build && cp bin/flowmi /opt/homebrew/bin/flowmi` to install the updated binary locally.

## Architecture

Cobra + Viper CLI. Entry: `main.go` → `cmd.Execute()`.

```
cmd/           Cobra commands — one file per resource (note.go, drive.go, email.go, etc.)
internal/
  auth/        OAuth2 PKCE generation, token exchange, local callback server
  api/         REST API client — all types + methods in client.go (envelope unwrapping)
  config/      XDG-compliant paths (~/.config/flowmi/), TOML config + credentials (0600)
  ui/          lipgloss terminal styles (TitleStyle, SuccessStyle, ErrorStyle, etc.)
```

### Command Tree

```
fm auth login|status     fm note list|create|view|edit|delete|trash|restore
fm drive list|upload|download|view|delete|trash|restore
fm table list|create|view|edit|delete|trash|restore
fm table field add|edit|delete                fm table row list|create|view|edit|delete|query|trash|restore
fm email send|list|view|delete|trash|restore  fm email mailbox list|create|edit|delete
fm search [web|images|news]                   fm scrape <url>
fm config set|get|list                        fm update | fm version | fm options
fm completion bash|zsh|fish|powershell
```

### Key Patterns

- **Two login flows**: browser OAuth2 PKCE (default) and direct password login (`--email`/`--password` for CI/CD). Both use PKCE.
- **Output format switch**: every display command uses the same `switch viper.GetString("output")` pattern with cases for `"json"`, `"table"`, `"text"/""`— follow this when adding commands.
- **`newAPIClient()` helper** (`cmd/note.go`): shared constructor that reads `access_token` from Viper, returns `*api.Client`, and wires up automatic token refresh on 401 (via `client.TokenRefresher`). Used by all authenticated commands.
- **API envelope**: server responses use `{"success": bool, "data": ..., "error": {"code": "...", "message": "..."}}`. The `api.Client.do()` method handles unwrapping.
- **Drive upload**: 3-step presigned URL flow — `InitUpload` → `UploadToPresignedURL` (PUT to R2) → `CompleteUpload`.
- **Binary alias**: supports both `flowmi` and `fm` — `cmd/root.go` adapts `Use` field based on `os.Args[0]`.
- **Structured errors**: `api.Error` carries code, message, hint, details, and requestID. `formatError()` in `cmd/root.go` renders errors in text or JSON based on `--output`. Exit codes map from error code prefixes: `AUTH_`→3, `NETWORK_`→4, `VALIDATION_`→2, `SERVER_`→5, default→1.
- **Config precedence**: flags → env vars (`FLOWMI_` prefix) → config.toml → credentials.toml defaults → hardcoded defaults (`flowmi.ai`, `api.flowmi.ai`).
- **Struct passing**: always pass structs by pointer (`*T`), not by value. This applies to function parameters, return values, and method receivers.
- **Vendored deps**: uses `vendor/` directory — run `go mod vendor` after adding/updating dependencies.

## JSON Convention

- All JSON field names use **camelCase** (e.g., `createdAt`, `userId`, `pageSize`, `requestId`).
- **Exception**: OAuth2 RFC 6749 protocol fields remain snake_case per spec (`client_id`, `redirect_uri`, `response_type`, `code_challenge`, `code_challenge_method`, `access_token`, `refresh_token`, `token_type`, `expires_in`, `grant_type`).
- This convention applies across all three Flowmi repos (server, web, CLI).

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

All Flowmi repos are under the **flowmi-ai** organization. The git remote uses the `humid` SSH alias (`git@humid:flowmi-ai/flowmi.git`). Run `gh auth switch` to the appropriate account before pushing, creating PRs, etc.

## Decision Making

When making technical decisions, follow community best practices over personal preference:
1. **Research first** — Search HN, Lobsters, SO, Reddit for real-world discussions before deciding
2. **Don't repeat known mistakes** — If the internet has lessons learned, don't rediscover them
3. **Be honest about consensus** — If a direction conflicts with community consensus, say so clearly
