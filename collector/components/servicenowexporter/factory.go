package servicenowexporter

import (
	"context"
	"errors"
	"fmt"

	"github.com/lightstep/sn-collector/collector/servicenowexporter/internal/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

var errInvalidConfig = errors.New("invalid config for servicenowexporter")

func createMetricsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Metrics, error) {
	if err := component.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("cannot configure servicenow metrics exporter: %w", err)
	}
	oCfg := cfg.(*Config)
	me := newServiceNowProducer(set.Logger, oCfg)

	return exporterhelper.NewMetricsExporter(
		ctx,
		set,
		cfg,
		me.metricsDataPusher,
		// disable timeout
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(oCfg.BackOffConfig),
		exporterhelper.WithQueue(oCfg.QueueSettings),
		exporterhelper.WithShutdown(me.Close),
	)
}

func createLogsExporter(
	ctx context.Context,
	set exporter.CreateSettings,
	cfg component.Config,
) (exporter.Logs, error) {
	if err := component.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("cannot configure servicenow metrics exporter: %w", err)
	}
	oCfg := cfg.(*Config)
	me := newServiceNowProducer(set.Logger, oCfg)

	return exporterhelper.NewLogsExporter(
		ctx,
		set,
		cfg,
		me.logDataPusher,
		// disable timeout
		exporterhelper.WithTimeout(exporterhelper.TimeoutSettings{Timeout: 0}),
		exporterhelper.WithRetry(oCfg.BackOffConfig),
		exporterhelper.WithQueue(oCfg.QueueSettings),
		exporterhelper.WithShutdown(me.Close),
	)
}

func NewFactory() exporter.Factory {
	return exporter.NewFactory(
		component.MustNewType(metadata.Type),
		createDefaultConfig,
		exporter.WithMetrics(createMetricsExporter, metadata.MetricsStability),
		exporter.WithLogs(createLogsExporter, metadata.LogsStability),
	)
}
