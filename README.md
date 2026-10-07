# PostBoy

A keyboard-driven terminal HTTP/API client built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Features

- GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS
- Saved requests organised in collections, with unsaved-change tracking
- Per-request query params, headers, body and authorization
- Auth helpers: Bearer token, Basic auth, API key (header or query)
- Request body types (alt+t): Raw, URL-encoded form, multipart form with file fields, and binary file; Ctrl+O opens the raw body in `$EDITOR`
- Import curl commands (bash/cmd/PowerShell) and Postman v2.1 collections; copy any request as curl; export collections to Postman
- Environments editor with `{{VAR}}` interpolation, nested variables and built-ins (`{{$uuid}}`, `{{$timestamp}}`, ...)
- Secrets stored in the OS keychain, referenced as `{{ secret.NAME }}`
- Request history (SQLite) with sensitive headers redacted
- JSON pretty-printing and syntax highlighting (JSON, HTML, XML)
- Response tools: search, JSON path filter ([gjson syntax](https://github.com/tidwall/gjson/blob/master/SYNTAX.md)), raw view, copy, save to file; binary-safe
- Request timeout and cancellation; optional per-request `timeout_seconds`
- HTTP options in `config.json`: `insecure_skip_verify`, `disable_redirects`, `proxy_url`, `enable_cookies`

## Install

**Homebrew**

```sh
brew install dheeraj080/tap/postboy
```

**Go**

```sh
go install github.com/dheeraj080/PostBoy@latest
```

Or download a binary from the [releases page](https://github.com/dheeraj080/PostBoy/releases).

## Usage

```sh
postboy                     # start the TUI
postboy --data-dir ./.pb    # use a custom config/history directory
postboy import api.json     # import a Postman v2.1 collection
postboy --version           # print version
```

### Keybindings

| Key | Action |
|---|---|
| `Enter` (in URL) / `Alt+R` | Send request |
| `Alt+X` | Cancel request |
| `Ctrl+S` | Save request (asks for a name/collection the first time) |
| `Alt+S` | Save as a copy |
| `Alt+N` | New request (press twice to discard unsaved changes) |
| `Alt+O` | Open collections |
| `Alt+I` | Import a curl command or Postman collection file |
| `Alt+C` | Copy request as curl (secrets stay as `{{ secret.NAME }}`) |
| `Tab` / `Shift+Tab` | Cycle focus |
| `←/→` or `h/l` | Switch tabs (when a tab bar is focused) |
| `Alt+M` | Cycle HTTP method |
| `Alt+T` | Cycle request body type |
| `Ctrl+O` | Edit raw body in `$EDITOR` (Body tab focused) |
| `Alt+L` | Jump to headers |
| `Alt+H` | History |
| `Alt+E` | Cycle environment |
| `Alt+V` | Edit environments and variables |
| `Alt+K` | Secrets manager |
| `?` | Help (when not typing) |
| `Ctrl+C` | Quit |

In the Params/Headers lists: `n` new, `Enter` edit, `t` toggle, `d` delete.

In form fields (Body tab): `n` new, `Enter` edit, `t` toggle, `d` delete, `v` switch text/file (multipart).
File values in multipart/binary bodies are file paths.

In the response panel (Tab to focus it): `/` search (`n`/`N` next/previous), `f` filter JSON
(e.g. `data.#.id`, `items.#(price>10).name`), `r` raw/formatted, `y` copy, `s` save to file,
`Esc` clear search/filter. Copy and save use the filter result when a filter is active,
otherwise the raw body exactly as received.

In the Authorization tab: `←/→` change the auth type, `↑/↓` move, `Enter` edit a field.
Use `{{ secret.NAME }}` for credentials so they are kept in the OS keychain rather
than in plain text. Auth is applied at send time and never written to history.

In the Collections dialog: `Enter` open/expand, `n` new collection, `r` rename, `d d` delete,
`e` export to `<name>.postman_collection.json` in the current directory, `i` import.

In the Environments dialog (`Alt+V`): `←/→` switch environment, `a` make it active,
`N`/`R`/`D D` new/rename/delete environment; `n`/`Enter`/`d` add/edit/delete variables.
Variables can reference other variables (`BASE={{HOST}}/v1`) and secrets.

Built-in variables: `{{$uuid}}` (`$guid`, `$randomUUID`), `{{$timestamp}}`, `{{$timestampMs}}`,
`{{$isoTimestamp}}`, `{{$randomInt}}`, `{{$randomHex}}`.

## Data locations

Config and current draft (`config.json`), saved requests (`collections.json`) and
history (`postboy.db`) live in your user config directory:

- Linux: `~/.config/postboy/`
- macOS: `~/Library/Application Support/postboy/`
- Windows: `%AppData%\postboy\`

Secret values are stored only in the OS keychain. If no keychain is available, secrets are kept in memory and are lost on exit.

## Development

```sh
go test ./...
go build -o postboy .
```

Project layout:

| Path | Responsibility |
|---|---|
| `main.go` | Flags and program startup |
| `internal/config` | Config, request and auth schema; atomic load/save; migrations |
| `internal/collection` | Saved requests (`collections.json`) |
| `internal/fsutil` | Atomic file writes |
| `internal/secrets` | OS keychain with in-memory fallback |
| `internal/interp` | `{{VAR}}` / `{{$dynamic}}` / `{{ secret.NAME }}` expansion |
| `internal/importer` | curl parse/generate, Postman v2.1 import/export |
| `internal/httpclient` | Request building and execution |
| `internal/store` | SQLite history with versioned migrations |
| `internal/tui` | Bubble Tea UI (model, update, views, keymap) |

Releases are produced by GoReleaser when a `v*` tag is pushed.

## License

[MIT](LICENSE)
