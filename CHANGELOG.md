# Changelog

## v1.0.0

### Added
- Saved requests organized in collections with unsaved-change tracking.
- Request editor for URL, method, query params, headers, body, auth.
- Auth modes: none, Bearer, Basic, API key header/query.
- Body types: raw, URL-encoded form, multipart form with files, binary file.
- curl import/export and Postman v2.1 import/export.
- Environments, secrets, built-in `{{$uuid}}`-style variables.
- Headless runner `postboy run "<collection>" --env <env>`.
- Collection variables, `expected_status`, and response `captures` for extracting values.
- Search/filter/copy/save response body from the TUI.
- Race-safe tests, CI checks, and GoReleaser release pipeline.

### Changed
- TUI was rebuilt as a dual-pane request/response workspace.
- CLI package moved to `cmd/postboy` so installs produce the `postboy` binary.
- Removed deprecated GoReleaser Homebrew block.
- Headless runner exits non-zero on request errors, HTTP 4xx/5xx, and expected status mismatches.
