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
- [ ] Config file schema documented
- [ ] Clear platform-specific config dir
- [ ] Clear behavior when Keyring is unavailable
- [ ] Clear keyring/keychain setup docs
- [ ] Upgrade path from older config files
- [ ] Warnings for plain-text secrets
- [ ] Request proxy/timeout/TLS settings shown in UI or docs

## UI/UX
- [x] Compact two-pane layout
- [x] One-line command row
- [ ] Focused view for edit vs send vs response
- [ ] Consistent labels and keyboard shortcuts in footer/help
- [ ] Empty states that are useful, not huge empty frames
- [ ] Terminal resize handling
- [ ] Small width fallback
- [ ] Avoid too many borders/panels

## Reliability and Security
- [x] Sensitive headers are redacted in history
- [x] Secrets are not stored on disk by default
- [ ] Validate secrets do not leak in request output
- [ ] Safe handling of missing secret references
- [ ] No hard-coded credentials in repo
- [ ] `--insecure` warning/docs
- [ ] Proxy auth support
- [ ] Proxy schemes tested
- [ ] HTTP redirects tested
- [ ] HTTPS certificate behavior documented
- [ ] Large response display bounded
- [ ] File upload size limits sane

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
- [ ] README quick start
- [ ] Install instructions
- [ ] Environment/secret setup
- [ ] Import/export examples
- [ ] CI/cli usage
- [ ] Troubleshooting network, TLS, proxy, keyring
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