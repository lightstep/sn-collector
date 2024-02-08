FROM golang:1-bullseye AS build-env

RUN go install go.opentelemetry.io/collector/cmd/builder@v0.94.0

RUN apt-get update && apt-get install libpcap-dev -y

WORKDIR /otelcol
COPY ./components ./components
COPY otelcol-builder.yaml otelcol-builder.yaml

RUN builder --config  otelcol-builder.yaml

COPY ./otelcol.yaml /etc/otelcol/config.yaml

ENTRYPOINT ["/otelcol/otelcol-servicenow/otelcol-servicenow"]
CMD ["--config", "/etc/otelcol/config.yaml"]
