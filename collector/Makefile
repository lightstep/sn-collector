ALLDOC := $(shell find .. \( -name "*.md" -o -name "*.yaml" \) \
                                -type f | sort)

.PHONY: all - Default target
all: build

.PHONY: build - Build the collector
build: install-builder components/servicenowexporter/factory.go
	builder --config otelcol-builder.yaml

.PHONY: build-windows
build-windows: install-builder
	GOOS=windows GOARCH=amd64 builder --config otelcol-builder.yaml

.PHONY: build-linux
build-linux: install-builder
	GOOS=linux GOARCH=amd64 builder --config otelcol-builder.yaml

.PHONY: build-darwin
build-darwin: install-builder
	GOOS=darwin GOARCH=amd64 builder --config otelcol-builder.yaml

.PHONY: validate-linux
validate-linux:
	./otelcol-servicenow/otelcol-servicenow validate --config ./config/otelcol-linux-hostmetrics.yaml

.PHONY: validate-darwin
validate-darwin:
	./otelcol-servicenow/otelcol-servicenow validate --config ./config/otelcol-macos-hostmetrics.yaml

.PHONY: test - Run tests for servicenowexporter
test:
	cd components/servicenowexporter && go test -v ./...

.PHONY: install-tools
install-tools: install-builder
	go install github.com/mh-cbon/go-msi@latest

.PHONY: install-builder - Install the builder
install-builder:
	go install "go.opentelemetry.io/collector/cmd/builder@v0.102.1"

.PHONY: docker - Build docker image
docker:
	docker build . -t lightstep/servicenow-collector:latest
