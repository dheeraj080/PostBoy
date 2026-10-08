# PostBoy v1 Checklist

## Core Product Fit
- [x] Save requests in collections
- [x] Create/edit URL, headers, params, body, auth
- [x] Auth: no auth, bearer, basic, API key
- [x] Body modes: raw, urlencoded, multipart, file
- [x] Environments and variables
- [x] Secrets via OS keyring
- [x] History
- [x] Import curl/Postman
- [x] Export to curl/Postman
- [x] Response search/filter/copy/save
- [x] Configurable transport: timeout, TLS verify, redirects, proxy, cookies
- [x] Run a collection from CLI, e.g. `postboy run <collection> --env prod`
- [x] Collection-level variables/defaults
- [x] Response assertions / tests
- [ ] Pre- and post-request scripts/extract variables

## Installation
- [ ] `postboy --version` reports a release version from CI builds
- [x] `go install` installs a binary named `postboy`
- [ ] Publish builds for Windows/macOS/Linux
- [ ] Publish checksums
- [x] Add `postboy` to release assets/GoReleaser config correctly
- [ ] Document Homebrew/Wget/install scripts

## Config and Runtime
- [x] Config file schema documented
- [x] Clear platform-specific config dir
- [x] Clear behavior when Keyring is unavailable
- [x] Clear keyring/keychain setup docs
- [x] Upgrade path from older config files
- [x] Warnings for plain-text secrets
- [x] Request proxy/timeout/TLS settings shown in UI or docs

## UI/UX
- [x] Compact two-pane layout
- [x] One-line command row
- [ ] Focused view for edit vs send vs response
- [ ] Consistent labels and keyboard shortcuts in footer/help
- [x] Empty states that are useful, not huge empty frames
- [x] Terminal resize handling
- [x] Small width fallback
- [x] Avoid too many borders/panels

## Reliability and Security
- [x] Sensitive headers are redacted in history
- [x] Secrets are not stored on disk by default
- [x] Validate secrets do not leak in request output
- [x] Safe handling of missing secret references
- [x] No hard-coded credentials in repo
- [x] `--insecure` warning/docs
- [ ] Proxy auth support
- [ ] Persistent proxy schemes tested
- [x] HTTP redirects tested
- [x] HTTPS certificate behavior documented
- [x] Large response display bounded
- [x] File upload size limits sane

## Developer Experience
- [x] Go tests for most packages
- [x] Race tests for core packages
- [x] Staticcheck in CI
- [x] `go vet` in CI
- [x] `gofmt` CI gate
- [x] `go test -race ./...`
- [x] Add a quick smoke test for `postboy --help`
- [x] Release workflow tests binary

## Docs
- [x] README quick start
- [x] Install instructions
- [x] Environment/secret setup
- [x] Import/export examples
- [x] CI/cli usage
- [x] Troubleshooting network, TLS, proxy, keyring
- [ ] Screenshot/demo GIF
- [ ] Changelog
- [ ] Contributing notes

## Release Discipline
- [ ] Tag releases as `vX.Y.Z`
- [x] Version from git tags
- [x] GoReleaser skeleton is reviewed and actually produces assets
- [ ] `postboy v1.0.0` command and assets match
- [ ] Checksums published
- [ ] Announcements/docs updated

# v1 Acceptance Test
```powershell
go clean -cache
go test ./...
go run ./cmd/postboy --version
go run ./cmd/postboy --help
go run ./cmd/postboy import --help
```
Then manually:
- Build and install/ship a real `postboy` binary
- Open it, type a URL, send request
- Add headers/params/body/auth
- Save, reorder, export
- Import curl and Postman
- Switch environment and verify secret behavior
- Search/filter/copy/save a response