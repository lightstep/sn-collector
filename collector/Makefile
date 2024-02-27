ALLDOC := $(shell find .. \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

.PHONY: all - Default target
all: build

.PHONY: build - Build the collector
build: components/servicenowexporter/factory.go
	builder --config otelcol-builder.yaml

.PHONY: build-windows
build-windows:
	GOOS=windows GOARCH=amd64 builder --config otelcol-builder.yaml

.PHONY: build-linux
build-linux:
	GOOS=linux GOARCH=amd64 builder --config otelcol-builder.yaml

.PHONY: build-darwin
build-darwin:
	GOOS=darwin GOARCH=amd64 builder --config otelcol-builder.yaml

.PHONY: test - Run tests for servicenowexporter
test:
	cd components/servicenowexporter && go test -v ./...

.PHONY: install-tools
install-tools:
	go install go.opentelemetry.io/collector/cmd/builder@v0.95.0

.PHONY: docker - Build docker image
docker:
	docker build . -t lightstep/servicenow-collector:latest