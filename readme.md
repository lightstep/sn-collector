## ServiceNow OpenTelemetry Collector (Experimental)

<center>

[![Action Status](https://github.com/lightstep/sn-collector/workflows/Build/badge.svg)](https://github.com/lightstep/sn-collector/actions)
[![Action Test Status](https://github.com/lightstep/sn-collector/workflows/Tests/badge.svg)](https://github.com/lightstep/sn-collector/actions)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

</center>

> ⚠️ **Important**: This is pre-release, experimental software under active development. There will be breaking changes between releases and it has not been tested on all platforms. Please contact your ServiceNow account team before installing to review eligibility. **We recommend customers build their own collectors or use [OpenTelemetry Collector Contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib/) for production software.**

ServiceNow OpenTelemetry Collector is an experimental distribution of the [OpenTelemetry
Collector](https://github.com/open-telemetry/opentelemetry-collector). It
provides a unified way to receive, process, and export metric, trace, and log
data for [ServiceNow Cloud Observability](https://www.lightstep.com) and various services running on ServiceNow instances.

| Feature                                        | Status     | Docs                     |
| ---------------------------------------------- | ---------- | ------------------------ |
| Telemetry routing and processing ("gateway")   | 🟢          | 📘 [Community docs][14]  |
| Kubernetes cluster and workload monitoring     | 🛠️          | 📒 [Install guide][10]   |
| Linux server monitoring                        | 🟡          | 📒 [Install guide][11]   |
| Windows server monitoring                      | 🟡          | 📒 [Install guide][12]   |
| macOS monitoring                               | 🟡          | 📒 [Install guide][13]   |
| Remote management (opAMP)                      | 🛠️          | 📒 [Install guide][16]|
| HTTP synthetic checks                          | 🛠️          | 📘 [Community docs][15]  |

[10]: /docs/monitor-kubernetes.md
[11]: /docs/monitor-linux.md
[12]: /docs/monitor-windows.md
[13]: /docs/monitor-macos.md
[14]: https://opentelemetry.io/docs/collector/
[15]: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/httpcheckreceiver/documentation.md
[16]: /docs/remote-management.md

### Supported ServiceNow destinations

Native OTLP exporters can be used to send metrics, logs, and traces to ServiceNow Cloud Observability.

| Destination              | Metrics       | Logs             | Traces  | Events                 |
| ------------------------ | ------------- | ---------------- | ------  | ---------------------- |
| Cloud Observability      | OTLP          | OTLP             | OTLP    | OTLP (Logs)            |
| ServiceNow MID Server    | [Push API][6] | [HLA REST API][8]| -      | [Web Service API][7]   |
| ServiceNow Instance      | -             | -                | -       | [Web Service API][7]   |

[6]: https://docs.servicenow.com/bundle/vancouver-api-reference/page/integrate/inbound-rest/concept/push-metrics-MID-server.html
[7]: https://docs.servicenow.com/bundle/vancouver-it-operations-management/page/product/event-management/task/send-events-via-web-service.html
[8]: https://docs.servicenow.com/bundle/vancouver-it-operations-management/page/product/health-log-analytics-admin/task/hla-data-input-rest-api.html

### Supported ServiceNow sources

| Source                   | Metrics  | Logs                                 | Traces  | Events |
| ------------------------ | -------- | ------------------------------------ | ------- | ------ |
| ServiceNow Instance      | -        | [Log Export Service to OTLP Logs][5] | -       | -      |

[5]: https://support.servicenow.com/kb?id=kb_article_view&sysparm_article=KB1575051

### ServiceNow Collector Built-in Components

The following tables represent the supported components of the ServiceNow Collector. Our goal is to upstream all in-house developed components (marked with `*`), where possible, to the -contrib distribution of the OpenTelemetry Collector.

#### Receivers

| Receiver                                                         | Status                       |
| ---------------------------------------------------------------- | ---------------------------- |
| otlp                                                             | [contrib][1]                 | 
| prometheus                                                       | [contrib][1]                 |
| hostmetrics                                                      | [contrib][1]                 |
| kafka                                                            | [contrib][1]                 |
| k8sevents                                                        | [contrib][1]                 |
| k8sobjects                                                       | [contrib][1]                 |
| k8scluster                                                       | [contrib][1]                 |
| kubeletstats                                                     | [contrib][1]                 |
| filelog                                                          | [contrib][1]                 |
| httpcheck                                                        | [contrib][1]                 |
| `azuremonitor`*                                                  | [contrib][1]                 |
| azureeventhub                                                    | [contrib][1]                 |
| `osquery`*                                                       | [contrib][1]                 |

[1]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver

#### Processors

| Processor                                                        | Status               |
| ---------------------------------------------------------------- | -------------------- |
| k8sattributes                                                    | [contrib][2]         |
| resource                                                         | [contrib][2]         |
| resourcedetection                                                | [contrib][2]         |
| transform                                                        | [contrib][2]         |
| `concurrentbatch`*                                               | under development    |

[2]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor

#### Exporters

| Exporter                                                         | Status               |
| ---------------------------------------------------------------- | -------------------- |
| otlp                                                             | in core              |
| `servicenow`*                                                    | under development    |
| `arrow`*                                                         | under development    |
| debug                                                            | in core              |
| kafka                                                            | [contrib][3]         |

[3]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter

#### Extensions

| Extension                                                        | Status          |
| ---------------------------------------------------------------- | --------------- |
| healthcheck                                                      | [contrib][4]    |
| opamp                                                            | [contrib][4]    |

[4]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension

#### Connectors

| Extension                                                        | Status          |
| ---------------------------------------------------------------- | --------------- |
| countconnector                                                   | in [contrib][9] |

[9]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector

### Getting help

We are providing support via GitHub on a best effort basis. ServiceNow customers should also open a case on [https://support.servicenow.com/now](https://support.servicenow.com/now).

### Development and Contributing

* For contributing guidelines, refer to [CONTRIBUTING.md](CONTRIBUTING.md).
* For getting started with development, see our development docs at [docs/development.md](/docs/development.md).

### Acknowledgements

* Thank you to the many open-source distributions for OpenTelemetry collectors for providing patterns for deploying, building, and releasing this software. Where possible, we've tried to follow best practices and align with community standards and conventions.
