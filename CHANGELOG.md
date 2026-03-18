# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-03-02

### Added

- OAuth2 PKCE authentication with browser flow and direct password login for CI/CD
- Notes: create, list, view, edit, delete, trash, restore
- Drive: upload (presigned URL), download, list, view, delete
- Tables: create, list, view, edit, delete with field and row management
- Table row queries with filter, sort, aggregate, and group-by support
- Email: send, list, view, delete with mailbox management
- Web search (web, images, news) and web scraping
- Configuration management with XDG-compliant paths
- Multiple output formats: text, JSON, table
- Shell completion for bash, zsh, fish, and PowerShell
- Automatic token refresh on 401 responses
- Structured error handling with error codes, hints, and exit code mapping
- NO_COLOR and TERM=dumb support
- Credential masking in `fm config list`
- JSON help output (`--help --json`)
- GoReleaser configuration for cross-platform builds
- GitHub Actions CI/CD (test, lint, build, release)

[0.1.0]: https://github.com/humid888/flowmi/releases/tag/v0.1.0
