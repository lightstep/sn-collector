## resourcegraphconnector

Turns telemetry into metrics representing ServiceNow CMDB CI Classes.

WIP.

### usage

```
    # build
    make

    # run
    ./otelcol-dev/otelcol-dev --config otelcol.yaml

    # generate some fake telemetry with resource attributes
    telemetrygen metrics --duration 1s --otlp-insecure
```