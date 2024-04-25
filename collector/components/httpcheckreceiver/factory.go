// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package httpcheckreceiver // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/httpcheckreceiver"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/scraperhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/httpcheckreceiver/internal/metadata"
)

var errConfigNotHTTPCheck = errors.New("config was not a HTTP check receiver config")

// NewFactory creates a new receiver factory
func NewFactory() receiver.Factory {
	f := &httpcheckReceiverFactory{
		httpScrapers: make(map[*Config]*httpcheckScraper),
	}
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(f.createMetricsReceiver, metadata.MetricsStability),
		receiver.WithLogs(f.createLogsReceiver, metadata.LogsStability),
	)
}

type httpcheckReceiverFactory struct {
	httpScrapers map[*Config]*httpcheckScraper
}

func createDefaultConfig() component.Config {
	cfg := scraperhelper.NewDefaultControllerConfig()
	cfg.CollectionInterval = 60 * time.Second

	return &Config{
		ControllerConfig:     cfg,
		MetricsBuilderConfig: metadata.DefaultMetricsBuilderConfig(),
		Targets:              []*targetConfig{},
	}
}

func (factory *httpcheckReceiverFactory) ensureScraper(
	params receiver.CreateSettings,
	config component.Config) (*httpcheckScraper, error) {

	rconfig, ok := config.(*Config)
	if !ok {
		return nil, errConfigNotHTTPCheck
	}

	httpcheckScraper := factory.httpScrapers[rconfig]
	if httpcheckScraper != nil {
		return httpcheckScraper, nil
	}

	httpcheckScraper = newScraper(rconfig, params)
	factory.httpScrapers[rconfig] = httpcheckScraper
	return httpcheckScraper, nil
}

func (factory *httpcheckReceiverFactory) createMetricsReceiver(
	_ context.Context,
	params receiver.CreateSettings,
	rConf component.Config,
	consumer consumer.Metrics) (receiver.Metrics, error) {

	httpcheckScraper, err := factory.ensureScraper(params, rConf)
	if err != nil {
		return nil, err
	}

	cfg := rConf.(*Config)
	scraper, err := scraperhelper.NewScraper(metadata.Type.String(), httpcheckScraper.scrape, scraperhelper.WithStart(httpcheckScraper.start))
	if err != nil {
		return nil, err
	}

	return scraperhelper.NewScraperControllerReceiver(&cfg.ControllerConfig, params, consumer, scraperhelper.AddScraper(scraper))
}

func (factory *httpcheckReceiverFactory) createLogsReceiver(
	_ context.Context,
	params receiver.CreateSettings,
	rConf component.Config,
	consumer consumer.Logs) (receiver.Logs, error) {

	httpcheckScraper, err := factory.ensureScraper(params, rConf)
	if err != nil {
		return nil, err
	}

	httpcheckScraper.logs = consumer
	return &nopReceiver{}, nil
}

type nopReceiver struct {
}

func (receiver *nopReceiver) Start(_ context.Context, _ component.Host) error {
	return nil
}

func (receiver *nopReceiver) Shutdown(_ context.Context) error {
	return nil
}
