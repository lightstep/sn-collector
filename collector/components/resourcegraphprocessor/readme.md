## resourcegraphprocessor

Infers resources and resource relationships from metrics, logs and traces.

Uses redis for persistence.

### usage

```
    # build
    make

    # run
    ./otelcol-dev/otelcol-dev --config otelcol.yaml

    # generate some fake telemetry with resource attributes
    telemetrygen metrics --duration 1s --otlp-insecure
```