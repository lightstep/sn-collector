DEMO_NAME = "azure-eventhub-logs"
TAG = lightstep/servicenow-collector

.PHONY: all - Default target
all: build

.PHONY: build - Build the collector
build: components/servicenowexporter/factory.go
	builder --config otelcol-builder.yaml

.PHONY: test - Run tests for servicenowexporter
test:
	cd components/servicenowexporter && go test -v ./...

.PHONY: docker - Build docker image
docker-build:
	docker build . -t ${TAG}:latest

docker-run: docker-build
	docker run -d --name=${DEMO_NAME} -v ./:/conf \
		-e AZURE_EVENTHUB_CONNECTION=${AZURE_EVENTHUB_CONNECTION} \
		-e LS_ACCESS_TOKEN=${LS_ACCESS_TOKEN} \
		${TAG} \
		--config=/conf/collector.yaml
