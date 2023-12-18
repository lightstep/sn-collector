## Development

### Build


The OpenTelemetry Collector builder is required to build.

```sh
cd collector/
make
goreleaser release --snapshot --rm-dist
```

### Release

`goreleaser` is used to package multi-platform builds of the collector.

```sh
cd collector/

make install-tools # only needed the first build

make
goreleaser release --skip-sign --snapshot --rm-dist
```

To build for multiple platforms, this repository runs goreleaser automatically in a Github Action when a tag starting with `v` is pushed to the repossitory.

```sh
git tag v0.0.1
git push origin --tags
# ... Github Action to build and released kicked off remotely
```