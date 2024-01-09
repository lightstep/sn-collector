package resourcegraphconnector

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

const (
	typeStr   = "resourcegraphconnector"
	stability = component.StabilityLevelAlpha
)

func createDefaultConfig() component.Config {
	return &Config{}
}

// NewFactory returns a ConnectorFactory.
func NewFactory() connector.Factory {
	return connector.NewFactory(
		typeStr,
		createDefaultConfig,
		connector.WithMetricsToMetrics(createMetricsToMetrics, stability),
		connector.WithLogsToMetrics(createLogsToMetrics, stability),
	)
}

// createMetricsToMetrics creates a metrics to logs connector based on provided config.
func createLogsToMetrics(
	_ context.Context,
	set connector.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (connector.Logs, error) {
	config := cfg.(*Config)

	rs, err := config.loadResourceSchema()
	if err != nil {
		return nil, err
	}

	return &resource{
		metricsConsumer: nextConsumer,
		logger:          set.Logger,
		config:          config,
		resourceSchema:  rs,
		startTime:       pcommon.NewTimestampFromTime(time.Now()),
	}, nil
}

// createMetricsToMetrics creates a metrics to metrics connector based on provided config.
func createMetricsToMetrics(
	_ context.Context,
	set connector.CreateSettings,
	cfg component.Config,
	nextConsumer consumer.Metrics,
) (connector.Metrics, error) {
	config := cfg.(*Config)

	rs, err := config.loadResourceSchema()
	if err != nil {
		return nil, err
	}

	return &resource{
		metricsConsumer: nextConsumer,
		logger:          set.Logger,
		config:          config,
		resourceSchema:  rs,
		startTime:       pcommon.NewTimestampFromTime(time.Now()),
	}, nil
}
