# cratesio

Browse the crates.io Rust package registry

`cratesio` is a single pure-Go binary. It speaks to crates.io over plain
HTTPS, shapes the responses into clean records, and pipes into the rest of your
tools. No API key, nothing to run alongside it.

## Install

```bash
go install github.com/tamnd/cratesio-cli/cmd/cratesio@latest
```

Or grab a prebuilt binary from the [releases](https://github.com/tamnd/cratesio-cli/releases), or run
the container image:

```bash
docker run --rm ghcr.io/tamnd/cratesio:latest --help
```

## Usage

```bash
cratesio --help
cratesio search tokio
cratesio info serde
cratesio versions serde --limit 10
cratesio owners serde
cratesio deps serde
cratesio top
cratesio categories
cratesio keywords
```

## Development

```
cmd/cratesio/  thin main, wires cli.Root into fang
cli/           the cobra command tree
cratesio/      the library: HTTP client and data models
docs/          tago documentation site
```

```bash
make build      # ./bin/cratesio
make test       # go test ./...
make vet        # go vet ./...
```

## Releasing

Push a version tag and GitHub Actions runs GoReleaser, which builds the
archives, Linux packages, the multi-arch GHCR image, checksums, SBOMs, and a
cosign signature:

```bash
git tag v0.1.1
git push --tags
```

The Homebrew and Scoop steps self-disable until their tokens exist, so the first
release works with no extra secrets.

## License

Apache-2.0. See [LICENSE](LICENSE).
