# Contributing

## Run tests

```sh
go test ./...
```

## Format

```sh
gofmt -w .
```

## Vet / lint

```sh
go vet ./...
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
```

## Build locally

```sh
go build -o postboy.exe ./cmd/postboy
```

## Release

Use GoReleaser snapshots for local verification:

```sh
go run github.com/goreleaser/goreleaser/v2@latest release --snapshot --clean --skip=publish
```

Only tag a release when you want publish assets.
