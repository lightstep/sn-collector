// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package azuremonitorreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/azuremonitorreceiver/internal/metadata"
)

const (
	defaultCollectionInterval = 10 * time.Second
	defaultCloud              = azureCloud
)

var errConfigNotAzureMonitor = errors.New("Config was not a Azure Monitor receiver config")

// NewFactory creates a new receiver factory
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
		receiver.WithTraces(createTracesReceiver, metadata.TracesStability),
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
		// receiver.WithLogs(createLogsReceiver, metadata.LogsStability)
		// receiver.WithLogs(createLogsReceiver, component.StabilityLevelDevelopment)
	)
}

func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultScraperControllerSettings(metadata.Type)
	cfg.CollectionInterval = defaultCollectionInterval

	return &Config{
		ScraperControllerSettings:     cfg,
		MetricsBuilderConfig:          metadata.DefaultMetricsBuilderConfig(),
		CacheResources:                24 * 60 * 60,
		CacheResourcesDefinitions:     24 * 60 * 60,
		MaximumNumberOfMetricsInACall: 20,
		Services:                      monitorServices,
		Authentication:                servicePrincipal,
		Cloud:                         defaultCloud,
	}
}

func createMetricsReceiver(_ context.Context, params receiver.CreateSettings, rConf component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	cfg, ok := rConf.(*Config)
	if !ok {
		return nil, errConfigNotAzureMonitor
	}

	azureScraper := newScraper(cfg, params)
	scraper, err := scraperhelper.NewScraper(metadata.Type, azureScraper.scrape, scraperhelper.WithStart(azureScraper.start))
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(&cfg.ScraperControllerSettings, params, consumer, scraperhelper.AddScraper(scraper))
}

func createTracesReceiver(ctx context.Context, set receiver.CreateSettings, cfg component.Config, nextConsumer consumer.Traces) (receiver.Traces, error) {
	return nil, errors.New("traces receiver not supported")
}

func createLogsReceiver(ctx context.Context, set receiver.CreateSettings, cfg component.Config, nextConsumer consumer.Logs) (receiver.Logs, error) {
	return nil, errors.New("logs receiver not supported")
}

// type osQueryReceiver struct {
// 	config       *Config
// 	logger       *zap.Logger
// 	client       *osquery.ExtensionManagerClient
// 	logsConsumer consumer.Logs
// }

// func createLogsReceiver(context.Context, component.ReceiverCreateSettings, component.Receiver, consumer.Logs) (component.LogsReceiver, error) {
// 	return nil, errors.New("logs receiver not supported")
// }
