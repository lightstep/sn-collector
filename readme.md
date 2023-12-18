## sn-collector

ServiceNow-flavored OpenTelemetry collector experiments.

### data in
* `components/osqueryreceiver` turn osquery requests into OTLP logs

### data out
* `components/servicenowexporter` write metrics to MID servers

### data insights
* `components/resourcegraphprocessor` turn telemetry into CIs and CI relationships (stored in redis)
* `components/resourceapiextension` expose detected resources as an HTTP API
