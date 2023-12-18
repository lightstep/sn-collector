package servicenowexporter

import (
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configopaque"
	"go.opentelemetry.io/collector/config/configretry"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

type Config struct {
	exporterhelper.TimeoutSettings `mapstructure:",squash"` // squash ensures fields are correctly decoded in embedded struct.
	exporterhelper.QueueSettings   `mapstructure:"sending_queue"`
	configretry.BackOffConfig      `mapstructure:"retry_on_failure"`

	// PushMetricsURL is the full url of the ServiceNow instance to send push metrics to. Ex: http://127.0.0.1:8090/api/mid/sa/metrics
	PushMetricsURL string `mapstructure:"instance_metrics_url"`

	// PushLogsURL is the full url of the ServiceNow instance to send push logs to. Ex: http://127.0.0.1:8090/api/mid/hla/raw
	PushLogsURL string `mapstructure:"instance_logs_url"`

	// PushEventsURL is the full url of the ServiceNow instance to send push events to. Ex: http://127.0.0.1:8090/api/sn_em_connector/em/inbound_event?source=snotel
	PushEventsURL string `mapstructure:"instance_events_url"`

	// ApiKey is used to set an Authorization header with a bearer token
	ApiKey configopaque.String `mapstructure:"api_key"`

	// InsecureSkipVerify disables TLS certificate verification
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`

	// Username is used to optionally specify the basic auth username
	Username string `mapstructure:"username"`
	// Password is used to optionally specify the basic auth password
	Password configopaque.String `mapstructure:"password"`
}

func createDefaultConfig() component.Config {
	return &Config{
		PushMetricsURL:     "http://localhost:8090/api/mid/sa/metrics",
		PushLogsURL:        "",
		PushEventsURL:      "http://localhost:8090/api/sn_em_connector/em/inbound_event?source=snotel",
		InsecureSkipVerify: false,
		TimeoutSettings:    exporterhelper.NewDefaultTimeoutSettings(),
		BackOffConfig:      configretry.NewDefaultBackOffConfig(),
		QueueSettings:      exporterhelper.NewDefaultQueueSettings(),
	}
}
