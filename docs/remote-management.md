## Remote Management with the ServiceNow Collector

Version `0.11.0` and later of the ServiceNow Collector contain the [`opamp` extension](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/opampextension).

The opAMP extension can be used to connect to an opAMP server for remote management, including Cloud Observability.

More information about the opAMP remote management protocol is available on the [OpenTelemetry community site](https://opentelemetry.io/docs/specs/opamp/).

### Connecting to Cloud Observability

By default, most default configuration files in release packages will be configured for opAMP.  [special API Key](https://docs.lightstep.com/docs/create-and-manage-api-keys) is needed to connect to Cloud Observability's opAMP service.

The extension is configured via the `opamp` key under `extensions`. Below is an example:

```yaml
extensions:
  health_check:
  opamp:
    server:
      http:
        endpoint: https://opamp.lightstep.com/v1/opamp
        headers:
          "Authorization": "Bearer ${LS_OPAMP_API_KEY}"
```