.PHONY: all - Default target
all: build

.PHONY: build - Build the collector
build: components/servicenowexporter/factory.go
	builder --config otelcol-builder.yaml

.PHONY: test - Run tests for servicenowexporter
test:
	cd components/servicenowexporter && go test -v ./...

.PHONY: docker - Build docker image
docker:
	docker build . -t lightstep/servicenow-collector:latest