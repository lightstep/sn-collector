package servicenowexporter

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/pdata/plog"
)

const (
	typeStr   = "servicenowexporter"
	stability = component.StabilityLevelAlpha
)

var errInvalidConfig = errors.New("invalid config for tcpstatsreceiver")

type Config struct {
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createLogsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Logs, error) {
	if err := component.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("cannot configure servicenow logs exporter: %w", err)
	}
	return exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		func(_ context.Context, _ plog.Logs) error { return nil },
	)
}

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		typeStr,
		createDefaultConfig,
		exporter.WithLogs(createLogsExporter, stability),
	)
}
