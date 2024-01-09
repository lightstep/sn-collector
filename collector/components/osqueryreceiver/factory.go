package osqueryreceiver

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

const (
	typeStr       = "osqueryreceiver"
	defaultSocket = "/var/osquery/osquery.em"
	stability     = component.StabilityLevelAlpha
)

func createDefaultConfig() component.Config {
	return &Config{
		ExtensionsSocket: defaultSocket,
	}
}

func createLogsReceiver(
	ctx context.Context,
	set receiver.CreateSettings,
	cfg component.Config,
	consumer consumer.Logs,
) (receiver.Logs, error) {
	if err := component.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("cannot configure servicenow logs exporter: %w", err)
	}
	return &osQueryReceiver{
		config:       cfg.(*Config),
		logsConsumer: consumer,
		logger:       set.Logger,
	}, nil
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, stability),
	)
}
